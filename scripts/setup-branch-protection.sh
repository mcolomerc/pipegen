#!/bin/bash
# Script to set up GitHub branch protection rules via GitHub CLI

set -e

REPO="mcolomerc/pipegen"
BRANCH="main"

echo "🔒 Setting up GitHub branch protection for $REPO on branch $BRANCH"

# Check if gh CLI is available
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) is required but not installed."
    echo "📦 Install it from: https://cli.github.com/"
    exit 1
fi

# Check if user is authenticated
if ! gh auth status &> /dev/null; then
    echo "🔑 Please authenticate with GitHub CLI first:"
    echo "   gh auth login"
    exit 1
fi

echo "⚙️  Configuring branch protection rules..."

# Enable branch protection with required status checks
gh api repos/$REPO/branches/$BRANCH/protection \
    --method PUT \
    --field required_status_checks='{"strict":true,"contexts":["Lint","Test"]}' \
    --field enforce_admins=true \
    --field required_pull_request_reviews='{"required_approving_review_count":1,"dismiss_stale_reviews":true,"require_code_owner_reviews":false}' \
    --field restrictions=null \
    --field allow_force_pushes=false \
    --field allow_deletions=false

echo "✅ Branch protection rules configured for $BRANCH!"
echo ""
echo "📋 Protection rules applied:"
echo "   ✓ Require status checks (Lint, Test) to pass"
echo "   ✓ Require branches to be up to date"
echo "   ✓ Require pull request reviews (1 approver)"
echo "   ✓ Dismiss stale reviews on new commits"
echo "   ✓ Include administrators in restrictions"
echo "   ✓ Prevent force pushes"
echo "   ✓ Prevent branch deletion"
echo ""
echo "🎯 Now all pushes to main must:"
echo "   1. Pass golangci-lint checks"
echo "   2. Pass all tests"
echo "   3. Go through pull request review"
