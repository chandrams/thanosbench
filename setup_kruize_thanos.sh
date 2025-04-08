#!/bin/sh

export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

function get_date() {
	date "+%Y-%m-%d %H:%M:%S"
}

function time_diff() {
	ssec=`date --utc --date "$1" +%s`
	esec=`date --utc --date "$2" +%s`

	diffsec=$(($esec-$ssec))
	echo $diffsec
}

function check_running() {

	check_pod=$1
	check_pod_ns=$2
	kubectl_cmd="kubectl -n ${check_pod_ns}"

	echo "Info: Waiting for ${check_pod} to come up....."
	err_wait=0
	counter=0
	while true; do
		sleep 2
		pod_list=$(${kubectl_cmd} get pods | grep ${check_pod})
		pod_stat=$(echo "${pod_list}" | awk '{ print $3 }')
		if [[ -z "${pod_list}" ]]; then
		  echo "Error: No pods found matching ${check_pod}"
		  err=-1
		  break
		fi

		case "${pod_stat}" in
		"Completed")
			echo "Info: ${check_pod} deploy succeeded: ${pod_stat}"
			err=0
			break
			;;
		"OOMKilled")
			echo "Info: ${check_pod} deploy failed: ${pod_stat}"
			err=0
			exit -1
			;;
		"ContainerStatusUnknown")
			echo "Info: ${check_pod} deploy failed: ${pod_stat}"
			err=0
			break
			;;
		"Error")
			# On Error, wait for 10 seconds before exiting.
			err_wait=$((err_wait + 1))
			if [ ${err_wait} -gt 5 ]; then
				echo "Error: ${check_pod} deploy failed: ${pod_stat}"
				err=-1
				break
			fi
			;;
		*)
			sleep 3
			if [ $counter == 200 ]; then
				${kubectl_cmd} describe pod ${check_pod}
				echo "ERROR: ${check_pod} Pods failed to come up!"
				exit -1
			fi
			((counter++))
			;;
		esac
	done

	sleep 60
	${kubectl_cmd} get pods | grep ${check_pod}
	sleep 30
	${kubectl_cmd} get jobs
	echo
}

function usage() {
	echo
	echo "Usage: $0 [-p kruize profile] [-i thanosbench image] [-n number of clusters] [-t Max time for usage metrics] -s = skip thanos setup"
	echo " -p: Kruize profile defined in thanosbench [Default - kruize-15d-tiny]"
	echo " -i: Thanosbench image [Default - quay.io/chandra25ms/thanosbench:kruize]"
	echo " -n: No. of clusters [Default - 5]"
	echo " -o: No. of orgs [Default - 5]"
	echo " -t: Max time for usage metrics [Default - manifests/configmaps]"
	echo " -s: skip thanos setup"
	exit -1
}

profile="kruize-15d-tiny"
skip_setup=0
num_clusters=5
num_orgs=5
thanosbench_image="quay.io/chandra25ms/thanosbench:kruize"
maxtime=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
start_time=$(get_date)

while getopts p:s:i:n:o:t: gopts; do
	case ${gopts} in
	p)
		profile="${OPTARG}"
		;;
	s)
		skip_setup=1
		;;
	i)
		thanosbench_image="${OPTARG}"
		;;
	n)
		num_clusters="${OPTARG}"
		;;
	o)
		num_orgs="${OPTARG}"
		;;
	t)
		maxtime="${OPTARG}"
		;;
	[?])
		usage
		;;
	esac
done

echo
echo "****************************************************"
echo "Kruize Profile - $profile"
echo "Thanos bench image - $thanosbench_image"
echo "No. of orgs - $num_orgs"
echo "No. of clusters - $num_clusters"
echo "Max time - $maxtime"
echo "Skip Thanos Deployment - $skip_setup"
echo "****************************************************"
echo 

setup_dir="${PWD}/thanos_setup"

echo ""
echo "Create ${setup_dir} ..."
mkdir -p ${setup_dir}
echo ""

cd ${setup_dir}

echo ""
rm -rf thanosmark
echo "Clone thanosmark ${setup_dir} ..."
git clone -b kruize_env https://github.com/chandrams/thanosmark.git
echo ""

cd ${setup_dir}/thanosmark

echo ""
echo "Cleanup thanos setup ..."
oc delete jobs thanos-block-create -n thanos-bench
make teardown
echo ""

echo ""
echo "Creating namespace thanos-bench ..."
oc create namespace thanos-bench
echo ""

echo ""
echo "Setup minio object storage ..."
make objstore
echo ""

echo ""
echo "Setup thanos query store ..."
make query-store
echo ""

# Backup .env file
echo ""
echo "Backing up .env file ..."
cp .env .env.orig
echo ""


for (( j = 1; j <= ${num_orgs}; j++ ))
do
	for (( i = 1; i <= ${num_clusters}; i++ ))
	do

		# set the profile and cluster

		echo ""
		echo "Update the profile ..."
		sed -i s/PROFILE=kruize-15d-1k/PROFILE=${profile}/g .env
		sed -i s#THANOSBENCH_IMG=quay.io/chandra25ms/thanosbench:kruize#THANOSBENCH_IMG=${thanosbench_image}#g .env
		sed -i s/ORG=org-1/ORG=org-${j}/g .env
		sed -i s/CLUSTER=eu-1/CLUSTER=eu-${j}-${i}/g .env
		sed -i s/MAXTIME=30m/MAXTIME=${maxtime}/g .env
		sed -i s/WORKERS=6/WORKERS=20/g .env
		echo ""

		cat .env | grep CLUSTER
		sleep 5

		echo ""
		echo "Generate TSDB blocks and upload to thanos ..."
		make block-data
		echo ""

		echo ""
		echo "Check status of thanos-create-block pod ..."
		check_running "thanos-block-create" "thanos-bench"
		echo ""
		sleep 10 

		# Restore .env file
		echo ""
		echo "Restoring .env file ..."
		cp .env.orig .env
		echo ""

		oc delete jobs thanos-block-create -n thanos-bench
	done
done

end_time=$(get_date)
elapsed_time=$(time_diff "${start_time}" "${end_time}")
echo "Thanos setup completed, took ${elapsed_time} seconds"
