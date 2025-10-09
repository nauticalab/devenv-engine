#!/bin/bash
# Container startup script for developer environment: {{.Name}}
set -e

# === ENVIRONMENT SETUP ===
TARGET_UID={{.GetUserID}}
TARGET_GID={{.GetUserID}}
DEV_USERNAME="{{.Name}}"

# Path configuration
PYTHON_BIN_PATH="{{.PythonBinPath}}"
PYTHON_PATH="${PYTHON_BIN_PATH}/python3"
ENV_INIT_SCRIPT="/home/${DEV_USERNAME}/.devenv_init.sh"
ENV_BASH_SCRIPT="/home/${DEV_USERNAME}/.devenv_bash.sh"

echo "Starting container setup for user: ${DEV_USERNAME} (UID: ${TARGET_UID})"

# === SYSTEM PACKAGE INSTALLATION ===
echo "Installing core system packages..."
apt-get update
apt-get install -y sudo openssh-server

# Install Homebrew dependencies if Homebrew will be installed
{{- if .InstallHomebrew}}
echo "Installing Homebrew dependencies"
apt-get install -y curl git build-essential file procps ca-certificates
{{- end}}

echo "Section 1: Environment and system setup complete"

# === USER MANAGEMENT ===
echo "Setting up user: ${DEV_USERNAME}"

# Create/rename group with target GID
if id -g ${TARGET_GID} &>/dev/null; then
    echo "Renaming group ${TARGET_GID} to ${DEV_USERNAME}"
    groupmod -n ${DEV_USERNAME} $(id -gn ${TARGET_GID})
else
    echo "Adding group ${DEV_USERNAME} with GID ${TARGET_GID}"
    groupadd -g ${TARGET_GID} ${DEV_USERNAME}
fi

# Create/rename user with target UID
if id -u ${TARGET_UID} &>/dev/null; then
    echo "Renaming user ${TARGET_UID} to ${DEV_USERNAME}"
    usermod -l ${DEV_USERNAME} -s /bin/bash -d /home/${DEV_USERNAME} -g ${TARGET_GID} $(id -un ${TARGET_UID})
else
    echo "Adding user ${DEV_USERNAME} with UID ${TARGET_UID}"
    useradd -u ${TARGET_UID} -m -s /bin/bash ${DEV_USERNAME}
fi

# Ensure home directory exists and has correct ownership
mkdir -p "/home/${DEV_USERNAME}"
chown ${DEV_USERNAME}:${DEV_USERNAME} "/home/${DEV_USERNAME}"

echo "Section 2: User management complete"

# === ADMIN PRIVILEGES ===
{{- if .IsAdmin}}
echo "Setting up admin privileges for ${DEV_USERNAME}"
usermod -aG sudo ${DEV_USERNAME}

# Configure sudo to not require password
echo "${DEV_USERNAME} ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/${DEV_USERNAME}
chmod 440 /etc/sudoers.d/${DEV_USERNAME}
{{- else}}
echo "User ${DEV_USERNAME} configured as non-admin"
{{- end}}

echo "Section 3: Admin privileges complete"

# === HOMEBREW INSTALLATION ===
{{- if .InstallHomebrew}}
echo "Installing Homebrew for ${DEV_USERNAME}"

# Create a specific sudoers file for Homebrew installation
echo "${DEV_USERNAME} ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/homebrew_install
chmod 440 /etc/sudoers.d/homebrew_install

# Install Homebrew as the dev user
sudo -u ${DEV_USERNAME} bash -c 'NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'

# Remove the temporary sudoers file
rm -f /etc/sudoers.d/homebrew_install

# Fix potential permissions issues
chown -R ${DEV_USERNAME}:${DEV_USERNAME} /home/${DEV_USERNAME}/.cache
{{- else}}
echo "Skipping Homebrew installation (disabled in config)"
{{- end}}

echo "Section 4: Homebrew installation complete"

# === SSH SERVER SETUP ===
echo "Setting up SSH server"

# Install and configure SSH server
mkdir -p /run/sshd

# Set up SSH host keys with persistence
mkdir -p /ssh_host_keys
if [ -f /ssh_host_keys/ssh_host_rsa_key ]; then
    # Use existing host keys
    cp /ssh_host_keys/ssh_host_* /etc/ssh/
else
    # Generate new host keys and store them
    ssh-keygen -A
    cp /etc/ssh/ssh_host_* /ssh_host_keys/
fi

# Ensure correct permissions on host keys
chmod 600 /etc/ssh/ssh_host_*
chmod 644 /etc/ssh/ssh_host_*.pub

# Set up SSH authorized keys for the developer
mkdir -p /home/${DEV_USERNAME}/.ssh
echo "{{.GetSSHKeysString}}" > /home/${DEV_USERNAME}/.ssh/authorized_keys
chmod 700 /home/${DEV_USERNAME}/.ssh
chmod 600 /home/${DEV_USERNAME}/.ssh/authorized_keys
chown -R ${DEV_USERNAME}:${DEV_USERNAME} /home/${DEV_USERNAME}/.ssh

echo "Section 5: SSH server setup complete"

# === PACKAGE INSTALLATION ===
{{- if gt (len .Packages.APT) 0}}
echo "Installing APT packages: {{range $i, $pkg := .Packages.APT}}{{if gt $i 0}} {{end}}{{$pkg}}{{end}}"
apt-get install -y{{range .Packages.APT}} {{.}}{{end}}
{{- end}}

{{- if .ClearLocalPackages}}
# Clear local packages if specified
echo "Clearing local packages"
rm -rf /home/${DEV_USERNAME}/.cache/pip
rm -rf /home/${DEV_USERNAME}/.local/lib/python*/site-packages/*
{{- end}}

# Install common python packages from requirements.txt
if [ -f /scripts/requirements.txt ]; then
    echo "Installing Python packages from requirements.txt"
    /bin/bash /scripts/run_with_git.sh ${DEV_USERNAME} ${PYTHON_PATH} -m pip install --no-user --no-cache-dir -r /scripts/requirements.txt
fi

{{- if gt (len .Packages.Python) 0}}
echo "Installing Python packages: {{range $i, $pkg := .Packages.Python}}{{if gt $i 0}} {{end}}{{$pkg}}{{end}}"
/bin/bash /scripts/run_with_git.sh ${DEV_USERNAME} ${PYTHON_PATH} -m pip install --no-user --no-cache-dir{{range .Packages.Python}} {{.}}{{end}}
{{- end}}

echo "Section 6: Package installation complete"

# === USER ENVIRONMENT SETUP ===
# Set up environment for the user
if [ -f /scripts/setup.sh ]; then
    echo "Running user environment setup script"
    sudo -u ${DEV_USERNAME} \
        GIT_USER_NAME="{{.Git.Name}}" \
        GIT_USER_EMAIL="{{.Git.Email}}" \
        ENV_BASH_SCRIPT=${ENV_BASH_SCRIPT} \
        ENV_INIT_SCRIPT=${ENV_INIT_SCRIPT} \
        PYTHON_BIN_PATH=${PYTHON_BIN_PATH} \
        bash /scripts/setup.sh
fi

if [ -f "${ENV_INIT_SCRIPT}" ]; then
    echo "Running custom init script"
    if ! (set +e; sudo -u ${DEV_USERNAME} bash -c "cd /home/${DEV_USERNAME} && bash ${ENV_INIT_SCRIPT}"); then
        echo "Warning: init.sh script failed, but continuing startup..."
    fi
fi

echo "Section 7: User environment setup complete"

# === VSCODE CONFIGURATION ===
{{- if .ClearVSCodeCache}}
echo "Clearing VSCode server cache"
rm -rf /home/${DEV_USERNAME}/.vscode-server/
{{- end}}


# Make sure .vscode-server directory is owned by ${DEV_USERNAME}
chown -R ${DEV_USERNAME}:${DEV_USERNAME} /home/${DEV_USERNAME}/.vscode-server

echo "Section 8: VSCode configuration complete"

# === SSH SERVER LAUNCH ===
echo "Starting SSH server"
/usr/sbin/sshd -D