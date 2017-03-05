#!/bin/bash

function loadInstanceIDs() {
  echo "declare -A instances"
  terraform show | egrep -o 'openvdc_instance.[^:]+:' | sed -e 's/openvdc_instance.\([^:]*\):/\1/g' | while read instance_name; do
    instance_id=`terraform show | grep -a1 "openvdc_instance.${instance_name}:" | tail -n1 | awk -F"= " '{print $2}'`
    echo "instances[${instance_name}]=${instance_id}"
  done
}

`loadInstanceIDs`

work_dir="/var/lib/jenkins/workspace/network.test"
testcase_dir="${work_dir}/{{.TestCaseDirectoryName}}"
result_file="${work_dir}/test_result.txt"
log_file="${work_dir}/test_log.txt"

\rm -f ${result_file} ${log_file}

cd ${testcase_dir}

find . -type f -name "*_server.sh" | while read server_script; do
  client_instance_name="`echo ${server_script} | awk -F_ '{print $1}' | sed -e 's/\.\///g'`"
  client_instance_id="${instances[${client_instance_name}]}"
  server_instance_name="`echo ${server_script} | awk -F_ '{print $3}'`"
  server_instance_id="${instances[${server_instance_name}]}"

  prefix="`echo ${server_script} | awk -F'_server' '{print $1}'`"
  echo "----- ${prefix} -----" | tee -a ${log_file}
  client_script="${prefix}_client.sh"
  chmod 755 ${server_script} ${client_script}

  if [ "" == "${server_instance_id}" ]; then
    server_log="skipped because the server is a physical device."
    server_script_result=0
  else
    openvdc wait ${server_instance_id} running
    server_log="`cat ${server_script} | openvdc console ${server_instance_id}`"
    server_script_result=$?
  fi

  echo "+++++ server_log +++++" | tee -a ${log_file}
  echo "${server_log}" | tee -a ${log_file}
  if [ ${server_script_result} -ne 0 ]; then
    echo "${prefix},NG" | tee -a ${result_file}
  else
    sleep 2

    openvdc wait ${client_instance_id} running
    client_log="`cat ${client_script} | openvdc console ${client_instance_id}`"
    client_script_result=$?

    echo "+++++ client_log +++++" | tee -a ${log_file}
    echo "${client_log}" | tee -a ${log_file}
    if [ ${client_script_result} -ne 0 ]; then
      echo "${prefix},NG" | tee -a ${result_file}
    else
      echo "${prefix},OK" | tee -a ${result_file}
    fi
  fi
  echo "---------------------" | tee -a ${log_file}
done
