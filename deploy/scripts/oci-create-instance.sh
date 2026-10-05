#!/usr/bin/env bash
# Runs ON YOUR PC (Git Bash) with the OCI CLI configured. Keeps trying to
# create the Always Free A1 instance until São Paulo has capacity.
#
#   export OCI_COMPARTMENT_ID=ocid1.tenancy.oc1..xxxx
#   export OCI_SUBNET_ID=ocid1.subnet.oc1.sa-saopaulo-1.xxxx
#   bash deploy/scripts/oci-create-instance.sh
#
# Optional: OCPUS (1), MEMORY_GB (6), SSH_PUBKEY_FILE, CLOUD_INIT_FILE,
# RETRY_SECONDS (90), IMAGE_ID (auto: latest Ubuntu 24.04 aarch64).
# 1 OCPU / 6 GB fits into capacity gaps more often; resize to 2 / 12
# later from the console (Edit > Shape). Both stay inside Always Free.
set -uo pipefail

: "${OCI_COMPARTMENT_ID:?export OCI_COMPARTMENT_ID (your tenancy OCID)}"
: "${OCI_SUBNET_ID:?export OCI_SUBNET_ID (public subnet OCID)}"
SSH_PUBKEY_FILE="${SSH_PUBKEY_FILE:-$HOME/.ssh/gtd_oci.pub}"
CLOUD_INIT_FILE="${CLOUD_INIT_FILE:-deploy/cloud-init.local.yaml}"
OCPUS="${OCPUS:-1}"
MEMORY_GB="${MEMORY_GB:-6}"
RETRY_SECONDS="${RETRY_SECONDS:-90}"
NAME=gtd-test
SHAPE=VM.Standard.A1.Flex

for f in "$SSH_PUBKEY_FILE" "$CLOUD_INIT_FILE"; do
  [ -f "$f" ] || { echo "missing file: $f"; exit 1; }
done
if grep -q '__[A-Z_]*__' "$CLOUD_INIT_FILE"; then
  echo "$CLOUD_INIT_FILE still has __PLACEHOLDERS__; fill them in first"; exit 1
fi

existing="$(oci compute instance list --compartment-id "$OCI_COMPARTMENT_ID" \
  --display-name "$NAME" --query "data[?\"lifecycle-state\"!='TERMINATED'].id | [0]" --raw-output 2>/dev/null)"
if [ -n "$existing" ] && [ "$existing" != "null" ]; then
  echo "instance $NAME already exists: $existing"; exit 0
fi

AD="$(oci iam availability-domain list --compartment-id "$OCI_COMPARTMENT_ID" \
  --query 'data[0].name' --raw-output)"
IMAGE_ID="${IMAGE_ID:-$(oci compute image list --compartment-id "$OCI_COMPARTMENT_ID" \
  --operating-system "Canonical Ubuntu" --operating-system-version "24.04" \
  --shape "$SHAPE" --sort-by TIMECREATED --sort-order DESC \
  --query 'data[0].id' --raw-output)}"

echo "AD:    $AD"
echo "image: $IMAGE_ID"
echo "shape: $SHAPE ${OCPUS} OCPU / ${MEMORY_GB} GB"

attempt=0
while true; do
  attempt=$((attempt + 1))
  out="$(oci compute instance launch \
    --compartment-id "$OCI_COMPARTMENT_ID" \
    --availability-domain "$AD" \
    --display-name "$NAME" \
    --shape "$SHAPE" \
    --shape-config "{\"ocpus\": $OCPUS, \"memoryInGBs\": $MEMORY_GB}" \
    --image-id "$IMAGE_ID" \
    --boot-volume-size-in-gbs 50 \
    --subnet-id "$OCI_SUBNET_ID" \
    --assign-public-ip true \
    --ssh-authorized-keys-file "$SSH_PUBKEY_FILE" \
    --user-data-file "$CLOUD_INIT_FILE" \
    --query 'data.id' --raw-output 2>&1)"
  status=$?

  if [ $status -eq 0 ]; then
    echo "$(date '+%F %T') created after $attempt attempt(s): $out"
    exit 0
  fi
  case "$out" in
    *"Out of host capacity"*|*TooManyRequests*|*InternalError*|*"timed out"*)
      echo "$(date '+%F %T') attempt $attempt: no capacity yet, retrying in ${RETRY_SECONDS}s"
      sleep "$RETRY_SECONDS"
      ;;
    *)
      echo "$(date '+%F %T') attempt $attempt failed with a non-capacity error:"
      echo "$out"
      exit 1
      ;;
  esac
done
