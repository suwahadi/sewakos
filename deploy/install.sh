#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/suwahadi/sewakos.git}"
BRANCH="${BRANCH:-main}"
APP_DIR="${APP_DIR:-/opt/sewakos}"
SWAP_SIZE="${SWAP_SIZE:-2G}"

log() { printf '\n==> %s\n' "$*"; }

if [ "$(id -u)" -ne 0 ]; then
  echo "install.sh: run as root, e.g. sudo bash install.sh" >&2
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive

log "Installing base packages"
apt-get update -qq
apt-get install -y -qq ca-certificates curl git ufw >/dev/null

if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
  log "Installing Docker Engine"
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  . /etc/os-release
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian ${VERSION_CODENAME} stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -qq
  apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin >/dev/null
  systemctl enable --now docker
fi

if [ -z "$(/usr/sbin/swapon --show --noheadings)" ]; then
  log "Adding ${SWAP_SIZE} swap (the frontend build needs more than 2 GB RAM)"
  fallocate -l "$SWAP_SIZE" /swapfile
  chmod 600 /swapfile
  /usr/sbin/mkswap /swapfile >/dev/null
  /usr/sbin/swapon /swapfile
  grep -q '^/swapfile ' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
fi

log "Configuring firewall (SSH, HTTP, HTTPS, storage)"
ufw allow OpenSSH >/dev/null
ufw allow 80/tcp >/dev/null
ufw allow 443/tcp >/dev/null
ufw allow 9000/tcp >/dev/null
ufw --force enable >/dev/null

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$script_dir/docker-compose.yml" ] && [ -d "$script_dir/../backend" ]; then
  APP_DIR="$(cd "$script_dir/.." && pwd)"
elif [ -d "$APP_DIR/.git" ]; then
  log "Updating $APP_DIR"
  git -C "$APP_DIR" fetch --quiet origin "$BRANCH"
  git -C "$APP_DIR" checkout --quiet "$BRANCH"
  git -C "$APP_DIR" merge --ff-only --quiet "origin/$BRANCH"
else
  log "Cloning $REPO_URL into $APP_DIR"
  git clone --quiet --branch "$BRANCH" "$REPO_URL" "$APP_DIR"
fi
cd "$APP_DIR/deploy"

random() { od -An -tx1 -N "$(($1 / 2))" /dev/urandom | tr -d ' \n'; }

if [ ! -f .env ]; then
  public_host="${PUBLIC_HOST:-$(ip -4 route get 1.1.1.1 | awk '{for (i = 1; i < NF; i++) if ($i == "src") print $(i + 1)}')}"
  acme_profile=classic
  www_host="www.$public_host"
  if [[ "$public_host" =~ ^[0-9.]+$ ]]; then
    acme_profile=shortlived
    www_host=localhost
  fi
  log "Generating deploy/.env for $public_host"
  umask 077
  cat > .env <<EOF
PUBLIC_HOST=$public_host
ACME_PROFILE=$acme_profile
WWW_HOST=$www_host
POSTGRES_PASSWORD=$(random 40)
APP_DB_PASSWORD=$(random 40)
STORAGE_ACCESS_KEY=sewakos
STORAGE_SECRET_KEY=$(random 40)
API_CLIENT_ID=SEWAKOS-WEB-$(random 24)
MIDTRANS_SERVER_KEY=
MIDTRANS_CLIENT_KEY=
MIDTRANS_SIGNATURE_KEY=
MIDTRANS_IS_PRODUCTION=false
SMTP_HOST=
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
EMAIL_FROM="Sewa Kos <no-reply@$public_host>"
EOF
fi
set -a
. ./.env
set +a

log "Building images"
docker compose build --pull

log "Starting database, cache and storage"
docker compose up -d --wait postgres redis minio
docker compose run --rm minio-init

log "Running migrations"
docker compose run --rm migrate
docker compose exec -T postgres psql -q -U kos -d kos -v ON_ERROR_STOP=1 \
  -c "ALTER ROLE kos_app WITH PASSWORD '${APP_DB_PASSWORD}'"

log "Starting the application"
docker compose up -d --remove-orphans
docker image prune -f >/dev/null
docker builder prune -f --keep-storage 5GB >/dev/null

log "Done: https://${PUBLIC_HOST}"
cat <<EOF
Create the first admin (choose your own name, phone and password):
  cd $APP_DIR/deploy && sudo docker compose run --rm api admin create --name "Admin" --phone 08xxxxxxxxxx --email admin@example.com --password '<password>'
Optionally load demo listings with Unsplash photos:
  cd $APP_DIR/deploy && sudo docker compose run --rm api demoseed --count 100
Follow logs (OTP codes are written there until an SMS/WhatsApp provider is set):
  cd $APP_DIR/deploy && sudo docker compose logs -f api worker
EOF
