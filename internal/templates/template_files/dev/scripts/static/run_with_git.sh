#!/bin/bash
#
# run_with_git - Run commands as another user with temporary Git credentials
#
# Usage: run_with_git.sh <username> <command> [arguments...]

set -e

if [ $# -lt 2 ]; then
  echo "Usage: $0 <username> <command> [arguments...]"
  exit 1
fi

# Get the target user and remove it from the arguments
TARGET_USER="$1"
shift

# Find the absolute path to git to ensure consistent behavior
GIT_PATH=$(which git)

# Build the command string preserving argument structure
CMD_STR=""
for arg in "$@"; do
  # Properly escape each argument
  CMD_STR="$CMD_STR \"${arg//\"/\\\"}\""
done

# Create the command to run as the other user
if [ -n "$GITHUB_TOKEN" ]; then
  # Create a temporary credentials file with secure permissions
  TEMP_CRED_FILE=$(mktemp -t git-cred-XXXXXX)
  chmod 600 "$TEMP_CRED_FILE"
  
  # Store GitHub token in credential file
  echo "https://x-access-token:$GITHUB_TOKEN@github.com" > "$TEMP_CRED_FILE"
  
  # Clean up the file regardless of how script exits
  trap 'rm -f "$TEMP_CRED_FILE"' EXIT
  
  sudo -u "$TARGET_USER" bash -c "
    # Save original Git credential helper
    ORIGINAL_HELPER=\$($GIT_PATH config --global credential.helper 2>/dev/null || echo '')
    
    # Set up file-based credential helper
    $GIT_PATH config --global credential.helper \"store --file=$TEMP_CRED_FILE\"
    
    # Clean up Git config on exit
    trap '$GIT_PATH config --global credential.helper \"\$ORIGINAL_HELPER\" 2>/dev/null || $GIT_PATH config --global --unset credential.helper' EXIT
    
    # Run the command
    $CMD_STR
  "
else
  # If GITHUB_TOKEN is not set, just run the command as the user
  echo "GITHUB_TOKEN not set, running command without git credential helper"
  sudo -u "$TARGET_USER" bash -c "$CMD_STR"
fi