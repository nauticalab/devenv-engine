#!/bin/bash
# User environment setup script for: {{.Name}}
# This script runs as the developer user to configure their personal environment
set -e

# Get script file names from environment
INIT_SCRIPT_NAME=$(basename "${ENV_INIT_SCRIPT}")
BASH_SCRIPT_NAME=$(basename "${ENV_BASH_SCRIPT}")

echo "Setting up user environment for {{.Name}}"

# === BASHRC SETUP ===
cat > ~/.bashrc << 'EOF_BASHRC'
{{- if .InstallHomebrew}}
# Set up Homebrew if installed
if [ -d "/home/linuxbrew/.linuxbrew" ]; then
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
fi
{{- end}}

# Add the Python bin path to the PATH
# Ensure this takes precedence over Homebrew
export PATH="{{.PythonBinPath}}:${PATH}"

# Custom aliases
alias ll='ls -la'

# GPU-related settings
export CUDA_DEVICE_ORDER=PCI_BUS_ID

# Run env bash script if it exists
if [ -f "${ENV_BASH_SCRIPT}" ]; then
  echo "Running ${ENV_BASH_SCRIPT}" >&2
  source ${ENV_BASH_SCRIPT}
fi

# Welcome message
echo "                                                      " >&2
echo "██████╗                  ███████╗ ██╗   ███╗ ██╗   ██╗" >&2
echo "██╔══██╗                 ╚════██║ ██║  ████║ ██║   ██║" >&2
echo "██║  ██║  ████╗ ██╗   ██╗  █████║ ██║ ██ ██║ ██║   ██║" >&2
echo "██║  ██║ ██▄▄▄█╗╚██╗ ██╔╝  ╚══██║ ██║██╔╝██║ ╚██╗ ██╔╝" >&2
echo "██████╔╝ ██▄▄▄▄╗ ╚████╔╝ ███████║ ████╔╝ ██║  ╚████╔╝ " >&2
echo "╚═════╝  ╚═════╝  ╚═══╝  ╚══════╝ ╚═══╝  ╚═╝   ╚═══╝  " >&2
echo "                                                      " >&2
echo "Welcome to your ENIGMA DevENV, {{.Name}}!" >&2
echo "Place ${INIT_SCRIPT_NAME} in your home directory to customize DevENV initialization." >&2
echo "Edit ${BASH_SCRIPT_NAME} to customize your Shell environment" >&2
{{- if .InstallHomebrew}}
if [ -d "/home/linuxbrew/.linuxbrew" ]; then
  echo "Homebrew is installed! Use 'brew' commands to install packages." >&2
fi
{{- end}}
echo "" >&2
echo "If you encounter issues, please contact the administrator." >&2
echo "Happy dev'ing!" >&2

EOF_BASHRC

# === BASH PROFILE SETUP ===
echo "source ~/.bashrc" > ~/.bash_profile

# === CREATE CUSTOM BASH SCRIPT ===
# Create env bash script file if it doesn't exist
if [ ! -f "${ENV_BASH_SCRIPT}" ]; then
  cat > "${ENV_BASH_SCRIPT}" << 'EOF_BASH_CUSTOM'
#!/bin/bash
# Custom bash environment configuration

# Add your custom environment variables and aliases here

{{- if .InstallHomebrew}}
# Homebrew-specific configurations can go here
if [ -d "/home/linuxbrew/.linuxbrew" ]; then
  # Add any brew-specific configurations here
  export HOMEBREW_NO_AUTO_UPDATE=1 # comment this line out to enable auto-update
fi
{{- end}}
EOF_BASH_CUSTOM
  chmod +x "${ENV_BASH_SCRIPT}"
fi

# === JUPYTER CONFIGURATION ===
mkdir -p ~/.jupyter
if [ ! -f ~/.jupyter/jupyter_notebook_config.py ]; then
cat > ~/.jupyter/jupyter_notebook_config.py << 'EOF_JUPYTER'
c.NotebookApp.ip = '0.0.0.0'
c.NotebookApp.open_browser = False
c.NotebookApp.port = 8888
EOF_JUPYTER
fi

# === GIT CONFIGURATION ===
{{- if and .Git.Name .Git.Email}}
echo "Configuring Git for {{.Name}}"
git config --global user.name "${GIT_USER_NAME}"
git config --global user.email "${GIT_USER_EMAIL}"
{{- end}}

echo "User environment setup complete for {{.Name}}"