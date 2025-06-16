package main

import (
    "encoding/csv"
    "flag"
    "fmt"
    "log"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "time"
)

type Service struct {
    PodNamePart string
    LocalPort   int
    ServicePort int
}

func parseArguments() (namespace, csvFile, service string, servicePort, localPort int, skipNoFound bool) {
    flag.StringVar(&namespace, "namespace", "", "The namespace of the Kubernetes pods.")
    flag.StringVar(&csvFile, "csv", "", "The CSV file containing the services to port forward.")
    flag.StringVar(&service, "service", "", "The name of the service to port forward.")
    flag.IntVar(&servicePort, "service-port", 0, "The service port to forward.")
    flag.IntVar(&localPort, "local-port", 0, "The local port to forward to.")
    flag.BoolVar(&skipNoFound, "skip-nofound", false, "If service not found should that be skip or raise error")
    flag.Parse()

    if namespace == "" {
        log.Fatal("--namespace is required")
    }
    if csvFile != "" && service != "" {
        log.Fatal("Only one of --csv or --service can be provided.")
    }
    if service != "" && (servicePort == 0 || localPort == 0) {
        log.Fatal("--service-port and --local-port are required when --service is provided.")
    }
    if csvFile == "" && service == "" {
        log.Fatal("Either --csv or --service must be provided.")
    }
    return
}

func getPodNames(namespace string) ([]string, error) {
    cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
    out, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    lines := strings.Split(strings.TrimSpace(string(out)), "\n")
    return lines, nil
}

func portForward(podNames []string, podNamePart string, localPort, servicePort int, namespace string, skipNoFound bool) {
    var podName string
    for _, name := range podNames {
        if strings.HasPrefix(name, podNamePart) {
            podName = name
            break
        }
    }
    if podName == "" {
        fmt.Printf("No pod found with name matching '%s' in namespace '%s'\n", podNamePart, namespace)
        if skipNoFound {
            return
        } else {
            log.Fatalf("Pod with name matching '%s' not found in namespace '%s'", podNamePart, namespace)
        }
    }
    fmt.Printf("Found pod: %s in namespace: %s\n", podName, namespace)
    fmt.Printf("Port forwarding from localhost:%d to %s:%d in namespace %s\n", localPort, podName, servicePort, namespace)
    cmd := exec.Command("kubectl", "port-forward", "-n", namespace, "pod/"+podName, fmt.Sprintf("%d:%d", localPort, servicePort))
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    err := cmd.Start()
    if err != nil {
        fmt.Printf("Error starting port-forward: %v\n", err)
    }
}

func readCSVAndPortForward(filename string) ([]Service, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        return nil, err
    }
    var services []Service
    header := make(map[string]int)
    for i, row := range records {
        if i == 0 {
            for idx, col := range row {
                header[col] = idx
            }
            continue
        }
        enabled := row[header["enabled"]]
        if enabled != "true" {
            continue
        }
        podNamePart := row[header["pod_name"]]
        localPort, _ := strconv.Atoi(row[header["local_port"]])
        servicePort, _ := strconv.Atoi(row[header["service_port"]])
        services = append(services, Service{podNamePart, localPort, servicePort})
    }
    return services, nil
}

func main() {
    namespace, csvFile, service, servicePort, localPort, skipNoFound := parseArguments()

    fmt.Println("Loading Kubernetes pods...")
    podNames, err := getPodNames(namespace)
    if err != nil {
        log.Fatalf("Failed to get pods: %v", err)
    }

    var services []Service
    if csvFile != "" {
        fmt.Println("Reading services.csv...")
        services, err = readCSVAndPortForward(csvFile)
        if err != nil {
            log.Fatalf("Failed to read CSV: %v", err)
        }
    } else if service != "" {
        fmt.Println("Using service details from arguments...")
        services = []Service{{service, localPort, servicePort}}
    }

    fmt.Printf("services_details: %+v\n", services)
    fmt.Println("Starting port forwarding...")
    for _, svc := range services {
        portForward(podNames, svc.PodNamePart, svc.LocalPort, svc.ServicePort, namespace, skipNoFound)
    }
    fmt.Println("Waiting for sometime")
    time.Sleep(5 * time.Second)
}