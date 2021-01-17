# Copyright 2017 Istio Authors
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

# x86 uses mysql:8.0.17 but no s390x compatible image instead use
# Need to instead build s390x compatible mysql image from source https://github.com/linux-on-ibm-z/dockerfile-examples/tree/master/MySQL/8.x
FROM localhost/mysql-s390x:8.0
# MYSQL_ROOT_PASSWORD must be supplied as an env var

COPY ./mysqldb-init.sql /docker-entrypoint-initdb.d
