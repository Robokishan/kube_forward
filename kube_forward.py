import csv
import subprocess
import time
import argparse


# list all port-forward
# ps -f | grep 'kubectl' | grep 'port-forward'
# stop all port-forward
# ps -f | grep 'kubectl' | grep 'port-forward' | grep -v grep | awk '{print $2}' | xargs kill

def parse_arguments():
    parser = argparse.ArgumentParser(description="Port forward to Kubernetes pods.")
    parser.add_argument("--namespace", type=str, required=True, help="The namespace of the Kubernetes pods.")
    
    parser.add_argument("--csv", type=str, help="The CSV file containing the services to port forward.")
    parser.add_argument("--service", type=str, help="The name of the service to port forward.")
    parser.add_argument("--service-port", type=int, help="The service port to forward.")
    parser.add_argument("--local-port", type=int, help="The local port to forward to.")
    
    args = parser.parse_args()

    if args.csv and args.service:
        parser.error("Only one of --csv or --service can be provided.")
    if args.service and (args.service_port is None or args.local_port is None):
        parser.error("--service-port and --local-port are required when --service is provided.")
    
    if not args.csv and not args.service:
        parser.error("Either --csv or --service must be provided.")
    
    return args

args = parse_arguments()
namespace = args.namespace
csv_file = args.csv
service = args.service
service_port = args.service_port
local_port = args.local_port


def port_forward(kube_lists,pod_name_part, local_port, service_port, namespace):
    # Find the pod name by matching the partial name within the specified namespace
    try:
        pod_name = next(
            (line for line in kube_lists.stdout.splitlines() if pod_name_part in line), None
        )
        if not pod_name:
            print(f"No pod found with name matching '{pod_name_part}' in namespace '{namespace}'")
            raise Exception(f"Pod with name matching '{pod_name_part}' not found in namespace '{namespace}'")

        print(f"Found pod: {pod_name} in namespace: {namespace}")
        # Start port-forwarding
        print(f"Port forwarding from localhost:{local_port} to {pod_name}:{service_port} in namespace {namespace}")
        print(f"kubectl", "port-forward", "-n", namespace, f"pod/{pod_name}", f"{local_port}:{service_port}")
        subprocess.Popen(
            ["kubectl", "port-forward", "-n", namespace, f"pod/{pod_name}", f"{local_port}:{service_port}"]
        )
        # time.sleep(2)  # Give it a moment to establish the connection

    except subprocess.CalledProcessError as e:
        print(f"Error: {e.stderr}")

def read_csv_and_port_forward(filename):
    services = []
    with open(filename, mode="r") as file:
        reader = csv.DictReader(file)
        for row in reader:
            pod_name_part = row["pod_name"]
            service_port = row["service_port"]
            local_port = row["local_port"]
            enabled = row["enabled"]
            if enabled == "true":
                services.append([pod_name_part,local_port ,service_port])
            # port_forward(pod_name_part, local_port, service_port, namespace)
    return services

print("Loading Kubernetes pods...")
kuberesults = subprocess.run(
            ["kubectl", "get", "pods", "-n", namespace, "--no-headers", "-o", "custom-columns=NAME:.metadata.name"],
            capture_output=True, text=True, check=True
        )
if csv_file:
    print("Reading services.csv...")
    services_details = read_csv_and_port_forward(csv_file)
elif service:
    print("Using service details from arguments...")
    services_details = [[service, local_port, service_port]]

print("services_details:",services_details)
print("Starting port forwarding...")
for service in services_details:
    port_forward(kuberesults, service[0], service[1], service[2], namespace)
print("Waiting for sometime")
time.sleep(5)