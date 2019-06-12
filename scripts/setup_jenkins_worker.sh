#!/bin/bash

set -ex

# Copyright 2019 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

sudo dnf update -y
sudo dnf install -y wget git java-1.8.0 docker gcc make openssl-devel libffi-devel bzip2-devel readline-devel sqlite-devel

sudo useradd -d /home/jenkins jenkins
sudo usermod -aG wheel jenkins

sudo groupadd docker
sudo usermod -aG docker jenkins

DOCKER_SOCKET=/var/run/docker.sock
DOCKER_GROUP=docker
DOCKER_GID=$(stat -c '%g' ${DOCKER_SOCKET})
sudo groupadd -for -g ${DOCKER_GID} ${DOCKER_GROUP}

sudo systemctl start docker
sudo systemctl enable docker

sudo chmod 666 /var/run/docker.sock

sudo su - jenkins
ssh-keygen -b 2048 -t rsa -f $HOME/.ssh/jenkins_id.rsa -q -N "" -C "yuanlin.xu@redhat.com"
cat $HOME/.ssh/jenkins_id.rsa.pub >> $HOME/.ssh/authorized_keys
# sudo vi /etc/pki/tls/cert.pem

mkdir -p $HOME/bin

git clone https://github.com/pyenv/pyenv.git $HOME/.pyenv

wget https://dl.google.com/go/go1.12.5.linux-amd64.tar.gz
tar -C $HOME/bin -xzf go1.12.5.linux-amd64.tar.gz
rm go1.12.5.linux-amd64.tar.gz

echo 'export PATH="$PATH:$HOME/.local/bin:$HOME/bin"' >> $HOME/.bashrc
echo 'export PYENV_ROOT="$HOME/.pyenv"' >> $HOME/.bashrc
echo 'export PATH="$PYENV_ROOT/bin:$PATH"' >> $HOME/.bashrc
echo -e 'if command -v pyenv 1>/dev/null 2>&1; then\n  eval "$(pyenv init -)"\nfi' >> $HOME/.bashrc
echo 'export GOPATH=$HOME/go' >> $HOME/.bashrc
echo 'export PATH="$PATH:$HOME/bin/go/bin:$GOPATH/bin"' >> $HOME/.bashrc
exec "$SHELL"

pyenv install 3.7.3
pyenv local 3.7.3
pyenv version
go version

exit

sudo ln -fs /home/jenkins/.pyenv/shims/python /usr/bin/python
