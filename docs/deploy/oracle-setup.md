# Setting up the GTD test VM on Oracle Cloud (Always Free)

From an empty Oracle account to automatic deploys on every push to `dev`.
Budget about two hours, plus however long São Paulo takes to have A1
capacity. Every resource here is **Always Free**: if a console screen
shows a price or lacks the "Always Free-eligible" label, stop and recheck.

Decisions behind this guide: [PLAN.md](PLAN.md).

Commands marked **(PC)** run in Git Bash on Windows. Commands marked
**(VM)** run over SSH on the VM.

---

## 1. Create two SSH keys (PC)

One key is for you to log in as `ubuntu`. The other is used only by
GitHub Actions to log in as `deploy`. Keeping them separate means you can
revoke CI access without locking yourself out.

```bash
ssh-keygen -t ed25519 -f ~/.ssh/gtd_oci -C "gtd admin"
ssh-keygen -t ed25519 -f ~/.ssh/gtd_deploy -C "gtd github-actions" -N ""
```

The deploy key has no passphrase because CI can't type one.

## 2. Create the network (Oracle console)

1. Sign in at <https://cloud.oracle.com>. Check that the region at the
   top right is **Brazil East (São Paulo)**.
2. Menu → **Networking → Virtual cloud networks → Start VCN Wizard →
   Create VCN with Internet Connectivity**.
3. Name it `gtd-vcn` and keep the defaults. Click **Next**, then **Create**.
4. Open `gtd-vcn` → **Subnets** → `public subnet-gtd-vcn`, and copy its
   **OCID**. You need it in step 6 if you use the retry script.

The default security list allows SSH (port 22) from anywhere and nothing
else. That's all you need: the app is never exposed publicly, and you
reach it through Tailscale.

> Oracle's Ubuntu images also ship a strict iptables firewall that only
> allows port 22. This setup doesn't open any other public port, so you
> don't need to change it. Tailscale adds its own rules for its traffic.

## 3. Set up Tailscale (browser + PC + phone)

1. Create a free account at <https://login.tailscale.com>. You can sign
   in with your GitHub account.
2. Install Tailscale on your **PC** and **phone**, and sign in on both.
3. **Access controls** → add tag owners to the policy file, then save:

   ```jsonc
   "tagOwners": {
     "tag:server": ["autogroup:admin"],
     "tag:ci":     ["autogroup:admin"],
   },
   ```

4. **DNS** → make sure **MagicDNS** is on, and enable **HTTPS Certificates**.
5. **Settings → Keys → Generate auth key**: turn off reusable, set the
   expiration to 1 day, turn on **Pre-approved**, turn on **Tags** and pick
   `tag:server`. Copy the `tskey-auth-...` value; it goes into the
   cloud-init file in step 4. Tagged nodes don't need their key renewed
   every 180 days.
6. **Settings → OAuth clients → Generate OAuth client**: give it the
   **Auth Keys → Write** scope with tag `tag:ci`. Save the **client ID**
   and **secret** for step 9.

## 4. Prepare the cloud-init file (PC)

```bash
cp deploy/cloud-init.yaml deploy/cloud-init.local.yaml
```

`*.local.yaml` is gitignored, so your secrets never get committed. In
`deploy/cloud-init.local.yaml`, replace:

| Placeholder | Value |
|---|---|
| `__DEPLOY_SSH_PUBKEY__` | the single line in `~/.ssh/gtd_deploy.pub` |
| `__TAILSCALE_AUTHKEY__` | the `tskey-auth-...` from step 3.5 |

## 5. Create the instance: try the console first

Menu → **Compute → Instances → Create instance**:

| Field | Value |
|---|---|
| Name | `gtd-test` |
| Image | **Canonical Ubuntu 24.04** (not "Minimal") |
| Shape | **Ampere → VM.Standard.A1.Flex**, **1 OCPU, 6 GB** |
| Networking | `gtd-vcn`, public subnet, **Assign a public IPv4 address** |
| SSH keys | **Paste public key**: contents of `~/.ssh/gtd_oci.pub` |
| Boot volume | default (≈47 GB) is fine; the free limit is 200 GB in total |
| Advanced options → Management | **Paste cloud-init script**: whole contents of `deploy/cloud-init.local.yaml` |

Click **Create**. If you get **"Out of host capacity"**, go to step 6.
Otherwise skip to step 7.

## 6. If there's no capacity: run the retry script (PC)

1. Install the OCI CLI. In PowerShell, follow the Windows quickstart at
   <https://docs.oracle.com/iaas/Content/API/SDKDocs/cliinstall.htm>.
   Then run `oci setup config` and accept the defaults. It creates an API
   key pair.
2. In the console: **Profile (top right) → My profile → API keys → Add API
   key → Paste public key**. Paste `~/.oci/oci_api_key_public.pem`.
3. Get your **tenancy OCID**: Profile → **Tenancy** → copy the OCID.
4. Run the script:

   ```bash
   export OCI_COMPARTMENT_ID=ocid1.tenancy.oc1..xxxx
   export OCI_SUBNET_ID=ocid1.subnet.oc1.sa-saopaulo-1.xxxx
   export SSH_PUBKEY_FILE=~/.ssh/gtd_oci.pub
   bash deploy/scripts/oci-create-instance.sh
   ```

   It tries every 90 s and stops on success or on any error that isn't
   about capacity. Leave it running. It can take hours or days. Keep the
   PC awake.

> The Tailscale auth key from step 3.5 expires after 1 day. If the script
> runs longer than that, generate a new key and update
> `cloud-init.local.yaml` before the instance gets created. Easiest: stop
> the script, update the key, start it again.

## 7. Check the first boot (PC → VM)

After about 5 minutes, `gtd-test` should appear in the Tailscale admin
**Machines** page. Then:

```bash
ssh -i ~/.ssh/gtd_oci ubuntu@gtd-test
```

On the VM:

```bash
cloud-init status --wait        # should end with "status: done"
docker compose version
swapon --show                   # /swapfile 2G
tailscale status
```

If something failed, read `/var/log/cloud-init-output.log`.

Publish the app inside your tailnet over HTTPS. This only needs doing once
per VM:

```bash
sudo tailscale serve --bg 8080
```

It prints the address, e.g. `https://gtd-test.<your-tailnet>.ts.net`.

## 8. Let the VM pull private images (GitHub + VM)

1. GitHub → **Settings → Developer settings → Personal access tokens →
   Tokens (classic) → Generate new token**. Select only **`read:packages`**
   and choose no expiration or a long one. Copy it.
2. On the VM:

   ```bash
   sudo -u deploy docker login ghcr.io -u gustavogmartinelli
   # paste the token as the password
   ```

## 9. Configure GitHub Actions (GitHub + PC)

From your PC (on the tailnet), capture the VM's SSH host key:

```bash
ssh-keyscan gtd-test 2>/dev/null
```

In the repo → **Settings → Environments → New environment → `test`**, then
add to the `test` environment:

| Kind | Name | Value |
|---|---|---|
| Secret | `TS_OAUTH_CLIENT_ID` | from step 3.6 |
| Secret | `TS_OAUTH_SECRET` | from step 3.6 |
| Secret | `DEPLOY_SSH_KEY` | whole contents of `~/.ssh/gtd_deploy` (the **private** key) |
| Variable | `DEPLOY_HOST` | `gtd-test` |
| Variable | `DEPLOY_KNOWN_HOSTS` | the full `ssh-keyscan` output |

Then in **Settings → Secrets and variables → Actions → Variables**
(repository level, *not* the environment), add `DEPLOY_ENABLED` =
`true`. The deploy job checks this before it runs, so it skips cleanly
until the VM exists.

## 10. First deploy

Merge or push to `dev`. In **Actions → Build and deploy**, the
`test → image → deploy-test` jobs should pass. The last step prints
`healthy on <sha>`.

Afterwards:

- Open `https://gtd-test.<your-tailnet>.ts.net/healthz` from your phone
  (with Tailscale on). It should return `{"status":"ok"}`.
- GitHub → your profile → **Packages → gtd → Package settings**: confirm
  the visibility is **Private**.

## 11. Backups (Oracle console + healthchecks.io + VM)

1. **Bucket:** Menu → **Storage → Buckets → Create bucket**. Name it
   `gtd-backups`, choose Standard tier, leave it private. On the bucket page,
   note the **Namespace**.
2. **S3 credentials:** Profile → **My profile → Customer secret keys →
   Generate secret key** (name `gtd-backup`). Copy the **secret** right
   away (it's shown once), then copy the **access key** from the list.
3. **Allow lifecycle rules:** Menu → **Identity & Security → Policies**,
   root compartment → **Create policy** `objectstorage-lifecycle` with:

   ```
   Allow service objectstorage-sa-saopaulo-1 to manage object-family in tenancy
   ```

4. **Retention:** bucket `gtd-backups` → **Lifecycle policy rules →
   Create rule**: name `daily-30d`, action **Delete**, **30 days**, object
   name filter → include prefix `daily/`. `monthly/` has no rule, so it's
   kept forever.
5. **Failure alerts:** sign up at <https://healthchecks.io> (free). Create
   a check `gtd-backup` with period **1 day** and grace **2 hours**, and
   copy its ping URL. If a night's backup fails or doesn't run, you get
   an email.
6. **On the VM**, configure rclone for root:

   ```bash
   sudo install -d -m 700 /root/.config/rclone
   sudo tee /root/.config/rclone/rclone.conf >/dev/null <<'EOF'
   [oci]
   type = s3
   provider = Other
   access_key_id = <ACCESS_KEY>
   secret_access_key = <SECRET>
   region = sa-saopaulo-1
   endpoint = https://<NAMESPACE>.compat.objectstorage.sa-saopaulo-1.oraclecloud.com
   force_path_style = true
   no_check_bucket = true
   EOF
   sudo chmod 600 /root/.config/rclone/rclone.conf
   sudo sed -i 's|^HC_PING_URL=.*|HC_PING_URL=https://hc-ping.com/<YOUR-UUID>|' /etc/gtd/backup.env
   ```

7. Run a backup now:

   ```bash
   sudo /opt/gtd/backup.sh
   sudo rclone ls oci:gtd-backups
   ```

   The healthchecks.io check should turn green. After that, cron runs it
   every night at 03:30 (São Paulo time). Logs go to
   `/var/log/gtd-backup.log`.

## 12. Restore drill (do this once)

A backup you've never restored isn't a backup yet.

```bash
# from your PC or phone, against https://gtd-test.<tailnet>.ts.net
curl -X POST https://gtd-test.<tailnet>.ts.net/items -d '{"title":"restore drill"}'
```

```bash
# on the VM
sudo /opt/gtd/backup.sh
# delete the item through the API, then:
sudo /opt/gtd/restore.sh                                  # lists backups
sudo /opt/gtd/restore.sh daily/gtd-$(date +%F).db.gz
```

The item is back. The database that was replaced stays next to it as
`/var/lib/gtd/gtd.db.before-restore-<timestamp>`. Delete it by hand
once you're satisfied.

---

## Day-to-day operations

| Task | How |
|---|---|
| Deploy | Push or merge to `dev` |
| Roll back | Actions → an older successful run on `dev` → **Re-run all jobs**. Or on the VM: edit `IMAGE_TAG` in `/opt/gtd/.env`, then `cd /opt/gtd && docker compose up -d` |
| Logs | `cd /opt/gtd && docker compose logs -f` |
| Status | `cd /opt/gtd && docker compose ps` (shows health) |
| Resize to 2 OCPU / 12 GB | Console → instance → **Edit → Edit shape**. The instance reboots |
| Add a schema change | New file `internal/adapter/repository/sqlite/migrations/0000N_<what>.sql`. It's applied on the next start. Never edit one that's already deployed |

## If Oracle reclaims or kills the VM

Free-tier-only accounts can lose instances that stay idle for 7 days.
Oracle emails a warning first. To rebuild:

1. Generate a new Tailscale auth key (step 3.5), update
   `cloud-init.local.yaml`, and remove the old `gtd-test` machine in the
   Tailscale admin.
2. Repeat steps 5–7, then step 8.
3. Update `DEPLOY_KNOWN_HOSTS` (step 9). The new VM has a new host key.
4. Repeat step 11.6 (rclone and healthchecks config).
5. Re-run the latest deploy on `dev`, then
   `sudo /opt/gtd/restore.sh daily/<latest>`.

If it happens more than once, reconsider upgrading to Pay As You Go
(Always Free resources stay free).

## Security notes

- The `deploy` user is in the `docker` group, which is root-equivalent.
  Protect `DEPLOY_SSH_KEY` accordingly. To cut off CI, remove its key from
  `/home/deploy/.ssh/authorized_keys`.
- Port 22 is public but accepts keys only. Once you're comfortable reaching
  the VM over Tailscale, you can remove the port-22 ingress rule from the
  security list.
- The app has **no authentication**. Keep it inside the tailnet (that is,
  never publish it with `tailscale funnel` or a public port) until auth
  exists.
