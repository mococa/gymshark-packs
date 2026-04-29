#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/setup-oidc.sh <github-username> <repo-name> [role-name]
#
# Sets up GitHub Actions OIDC trust in AWS so CI can assume an IAM role
# without long-lived access keys.
#
# Prerequisites: aws CLI configured with sufficient permissions (IAM write access).

GITHUB_USER="${1:?Usage: $0 <github-username> <repo-name> [role-name]}"
REPO="${2:?Usage: $0 <github-username> <repo-name> [role-name]}"
ROLE_NAME="${3:-github-actions-gymshark}"

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
OIDC_URL="token.actions.githubusercontent.com"
PROVIDER_ARN="arn:aws:iam::${ACCOUNT_ID}:oidc-provider/${OIDC_URL}"

echo "==> Deriving thumbprint from live TLS chain..."
THUMBPRINT=$(openssl s_client \
  -servername "${OIDC_URL}" \
  -connect "${OIDC_URL}:443" \
  -showcerts </dev/null 2>/dev/null \
| awk '/-----BEGIN CERTIFICATE-----/{cert=""} {cert=cert"\n"$0} /-----END CERTIFICATE-----/{last=cert} END{print last}' \
| openssl x509 -fingerprint -sha1 -noout \
| sed 's/.*=//' | tr -d ':' | tr 'A-F' 'a-f')
echo "    Thumbprint: ${THUMBPRINT}"

echo "==> Creating OIDC identity provider (skipping if already exists)..."
aws iam create-open-id-connect-provider \
  --url "https://${OIDC_URL}" \
  --client-id-list sts.amazonaws.com \
  --thumbprint-list "${THUMBPRINT}" 2>/dev/null \
  && echo "    Created." \
  || echo "    Already exists, skipping."

echo "==> Creating IAM role: ${ROLE_NAME}..."
TRUST_POLICY=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "${PROVIDER_ARN}"
    },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": {
        "${OIDC_URL}:aud": "sts.amazonaws.com"
      },
      "StringLike": {
        "${OIDC_URL}:sub": "repo:${GITHUB_USER}/${REPO}:ref:refs/heads/main"
      }
    }
  }]
}
EOF
)

ROLE_ARN=$(aws iam create-role \
  --role-name "${ROLE_NAME}" \
  --assume-role-policy-document "${TRUST_POLICY}" \
  --query Role.Arn --output text)
echo "    Role ARN: ${ROLE_ARN}"

echo "==> Attaching AdministratorAccess..."
echo "    WARNING: This grants full AWS account access to your CI pipeline."
echo "    Acceptable for bootstrapping. Replace with a scoped policy before"
echo "    treating this account as anything beyond a personal project."
aws iam attach-role-policy \
  --role-name "${ROLE_NAME}" \
  --policy-arn arn:aws:iam::aws:policy/AdministratorAccess
echo "    Done."

echo ""
echo "Add these to your GitHub repository secrets:"
echo ""
echo "  AWS_ROLE_ARN = ${ROLE_ARN}"
echo ""
echo "Then run: make bootstrap"
echo "And add TF_STATE_BUCKET, TF_STATE_LOCK_TABLE, DOMAIN_NAME from its output."
