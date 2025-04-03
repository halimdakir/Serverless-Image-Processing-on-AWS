#!/bin/bash
endpoint="https://i3ceevb4v2.execute-api.us-east-1.amazonaws.com/dev"
payload_file="payload.json"
total_requests=1000
echo "=== Scale-Out Phase ==="
for conc in 5 10 20 30 40 50 100 150 200 250 300; do
  echo "Running ApacheBench with concurrency $conc"
  ab -n $total_requests -c $conc -T "application/json" -p $payload_file $endpoint > result_scale_out_$conc.txt
  sleep 60
done
echo "=== Scale-In Phase ==="
for conc in 300 250 200 150 100 50 40 30 20 10 5; do
  echo "Running ApacheBench with concurrency $conc"
  ab -n $total_requests -c $conc -T "application/json" -p $payload_file $endpoint > result_scale_in_$conc.txt
  sleep 60
done
echo "Test completed. Analyze results in result_scale_out_*.txt, result_scale_in_*.txt"