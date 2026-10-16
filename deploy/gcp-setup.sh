#!/usr/bin/env bash
# One-time GCP setup for CI/CD to Cloud Run. Safe to re-run.
#
#   PROJECT_ID=my-project GITHUB_REPO=sianwa11/shazam-mwitu ./deploy/gcp-setup.sh
#
set -euo pipefail

: "${PROJECT_ID:?set PROJECT_ID}"
: "${GITHUB_REPO:?set GITHUB_REPO, e.g. sianwa11/shazam-mwitu}"
REGION="${REGION:-europe-west1}" # africa-south1 (Johannesburg) is also available
BUCKET="${BUCKET:-${PROJECT_ID}-shazam-db}"
AR_REPO="shazam-mwitu"
POOL="github"
PROVIDER="github-provider"

gcloud config set project "$PROJECT_ID" >/dev/null
PROJECT_NUMBER="$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')"
DEPLOY_SA="gh-deployer@${PROJECT_ID}.iam.gserviceaccount.com"
RUNTIME_SA="shazam-runtime@${PROJECT_ID}.iam.gserviceaccount.com"

echo "Enabling APIs..."
gcloud services enable \
  run.googleapis.com \
  artifactregistry.googleapis.com \
  iam.googleapis.com \
  iamcredentials.googleapis.com \
  sts.googleapis.com \
  cloudresourcemanager.googleapis.com \
  storage.googleapis.com

echo "Artifact Registry repository..."
gcloud artifacts repositories describe "$AR_REPO" --location="$REGION" >/dev/null 2>&1 ||
  gcloud artifacts repositories create "$AR_REPO" \
    --repository-format=docker --location="$REGION"

echo "Storage bucket for songs.db..."
gcloud storage buckets describe "gs://${BUCKET}" >/dev/null 2>&1 ||
  gcloud storage buckets create "gs://${BUCKET}" \
    --location="$REGION" --uniform-bucket-level-access

echo "Service accounts..."
for sa in gh-deployer shazam-runtime; do
  gcloud iam service-accounts describe "${sa}@${PROJECT_ID}.iam.gserviceaccount.com" >/dev/null 2>&1 ||
    gcloud iam service-accounts create "$sa"
done

echo "Granting the deployer what it needs..."
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${DEPLOY_SA}" \
  --role="roles/run.admin" --condition=None >/dev/null
gcloud artifacts repositories add-iam-policy-binding "$AR_REPO" --location="$REGION" \
  --member="serviceAccount:${DEPLOY_SA}" \
  --role="roles/artifactregistry.writer" >/dev/null
gcloud storage buckets add-iam-policy-binding "gs://${BUCKET}" \
  --member="serviceAccount:${DEPLOY_SA}" \
  --role="roles/storage.objectViewer" >/dev/null
gcloud iam service-accounts add-iam-policy-binding "$RUNTIME_SA" \
  --member="serviceAccount:${DEPLOY_SA}" \
  --role="roles/iam.serviceAccountUser" >/dev/null

echo "Workload Identity Federation (GitHub -> GCP, no stored keys)..."
gcloud iam workload-identity-pools describe "$POOL" --location=global >/dev/null 2>&1 ||
  gcloud iam workload-identity-pools create "$POOL" \
    --location=global --display-name="GitHub Actions"

gcloud iam workload-identity-pools providers describe "$PROVIDER" \
  --location=global --workload-identity-pool="$POOL" >/dev/null 2>&1 ||
  gcloud iam workload-identity-pools providers create-oidc "$PROVIDER" \
    --location=global --workload-identity-pool="$POOL" \
    --issuer-uri="https://token.actions.githubusercontent.com" \
    --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository" \
    --attribute-condition="assertion.repository == '${GITHUB_REPO}'"

gcloud iam service-accounts add-iam-policy-binding "$DEPLOY_SA" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL}/attribute.repository/${GITHUB_REPO}" >/dev/null

cat <<EOF

Done. In GitHub go to Settings > Secrets and variables > Actions > Variables
and add these repository variables (none of them are secrets):

  GCP_PROJECT_ID   = ${PROJECT_ID}
  GCP_REGION       = ${REGION}
  GCP_DB_BUCKET    = ${BUCKET}
  GCP_DEPLOY_SA    = ${DEPLOY_SA}
  GCP_RUNTIME_SA   = ${RUNTIME_SA}
  GCP_WIF_PROVIDER = projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL}/providers/${PROVIDER}

Then upload your fingerprint database:

  gcloud storage cp songs.db gs://${BUCKET}/songs.db
EOF