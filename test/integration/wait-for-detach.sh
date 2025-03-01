#!/bin/bash

# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

if [[ "$#" -lt 2 ]]; then
  echo "[Error] wrong number of arguments: the script requires two arguments: <Node Name> <Volume Name>"
  exit 1
fi

azVA="$1-$2-attachment"

until kubectl get azvolumeattachments $azVA -n azure-disk-csi 2>&1 | grep NotFound
do
    echo "Waiting for $azVA to be detached" 
    sleep 15
done

echo "$azVA successfully detached and deleted"