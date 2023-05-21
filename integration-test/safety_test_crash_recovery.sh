arrivalRate=$1
algo=$2
viewTimeoutTime=$3
batchTime=$4
batchSize=$5
pipelineLength=$6
window=$7

replica_path="replica/bin/replica"
ctl_path="client/bin/client"
output_path="logs/"

rm -r configuration/local
mkdir -p configuration/local
python3 configuration/config-generate.py 5 5 0.0.0.0 0.0.0.0 0.0.0.0 0.0.0.0 0.0.0.0 0.0.0.0 0.0.0.0 0.0.0.0 0.0.0.0 0.0.0.0 > configuration/local/configuration.yml

rm -r ${output_path}
mkdir ${output_path}

pkill replica; pkill replica; pkill replica
pkill client; pkill client; pkill client

echo "Killed previously running instances"

nohup ./${replica_path} --name 1 --consAlgo "${algo}" --batchSize "${batchSize}" --batchTime "${batchTime}"   --debugOn --debugLevel 100 --viewTimeout "${viewTimeoutTime}" --pipelineLength "${pipelineLength}" >${output_path}1.log &
nohup ./${replica_path} --name 2 --consAlgo "${algo}" --batchSize "${batchSize}" --batchTime "${batchTime}"   --debugOn --debugLevel 100 --viewTimeout "${viewTimeoutTime}" --pipelineLength "${pipelineLength}" >${output_path}2.log &
nohup ./${replica_path} --name 3 --consAlgo "${algo}" --batchSize "${batchSize}" --batchTime "${batchTime}"   --debugOn --debugLevel 100 --viewTimeout "${viewTimeoutTime}" --pipelineLength "${pipelineLength}" >${output_path}3.log &
nohup ./${replica_path} --name 4 --consAlgo "${algo}" --batchSize "${batchSize}" --batchTime "${batchTime}"   --debugOn --debugLevel 100 --viewTimeout "${viewTimeoutTime}" --pipelineLength "${pipelineLength}" >${output_path}4.log &
nohup ./${replica_path} --name 5 --consAlgo "${algo}" --batchSize "${batchSize}" --batchTime "${batchTime}"   --debugOn --debugLevel 100 --viewTimeout "${viewTimeoutTime}" --pipelineLength "${pipelineLength}" >${output_path}5.log &

echo "Started 5 replicas"

sleep 5

./${ctl_path} --name 21 --requestType status --operationType 1  --debugOn --debugLevel 0 >${output_path}status1.log

sleep 5

echo "sent initial status"

./${ctl_path} --name 22 --requestType status --operationType 3  --debugOn --debugLevel 0 >${output_path}status3.log

sleep 5

echo "sent consensus start up status"

echo "starting clients"

nohup ./${ctl_path} --name 21 --requestType request --debugOn --debugLevel 100 --batchSize  "${batchSize}" --batchTime "${batchTime}" --arrivalRate "${arrivalRate}" --window "${window}" >${output_path}21.log &
nohup ./${ctl_path} --name 22 --requestType request --debugOn --debugLevel 100 --batchSize  "${batchSize}" --batchTime "${batchTime}" --arrivalRate "${arrivalRate}" --window "${window}" >${output_path}22.log &
nohup ./${ctl_path} --name 23 --requestType request --debugOn --debugLevel 100 --batchSize  "${batchSize}" --batchTime "${batchTime}" --arrivalRate "${arrivalRate}" --window "${window}" >${output_path}23.log &
nohup ./${ctl_path} --name 24 --requestType request --debugOn --debugLevel 100 --batchSize  "${batchSize}" --batchTime "${batchTime}" --arrivalRate "${arrivalRate}" --window "${window}" >${output_path}24.log &
nohup ./${ctl_path} --name 25 --requestType request --debugOn --debugLevel 100 --batchSize  "${batchSize}" --batchTime "${batchTime}" --arrivalRate "${arrivalRate}" --window "${window}" >${output_path}25.log &

sleep 120

echo "finished running clients"


nohup ./${ctl_path} --name 21 --requestType status --operationType 2  --debugOn --debugLevel 10 >${output_path}status2.log &


echo "sent status to print logs"

sleep 30

pkill replica; pkill replica; pkill replica; pkill replica; pkill replica
pkill client; pkill client; pkill client; pkill client; pkill client
rm -r configuration/local

python3 experiments/python/overlay-test.py logs/1-consensus.txt logs/2-consensus.txt logs/3-consensus.txt logs/4-consensus.txt logs/5-consensus.txt > ${output_path}python-consensus.log

echo "Killed instances"

echo "Finish test"
