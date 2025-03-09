package test

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

// 解析 JSON 并转换成 []corev1.ServicePort
func parsePorts(portJson string) ([]corev1.ServicePort, error) {
	T := "ports:\n  - containerPort: 80\n    protocol: TCP\n"
	fmt.Println(T)
	fmt.Println(portJson)
	portJson = T
	// 解析 YAML 数据到 map
	var rawData struct {
		Ports []struct {
			ContainerPort int32  `yaml:"containerPort"`
			Protocol      string `yaml:"protocol"`
		} `yaml:"ports"`
	}

	if err := yaml.Unmarshal([]byte(portJson), &rawData); err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML: %v", err)
	}

	// 转换到 corev1.ServicePort
	var servicePorts []corev1.ServicePort
	for _, p := range rawData.Ports {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Port:     p.ContainerPort,
			Protocol: corev1.Protocol(p.Protocol),
		})
	}

	return servicePorts, nil
}
func Test1(t *testing.T) {
	// 修正 YAML 格式，确保正确缩进
	jsonInput := `{
		"port_json": "ports:\n  - containerPort: 80\n    protocol: TCP\n"
	}`

	// 解析 JSON 获取 port_json
	var inputData struct {
		PortJson string `json:"port_json"`
	}
	if err := json.Unmarshal([]byte(jsonInput), &inputData); err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	// 解析 PortJson
	servicePorts, err := parsePorts(inputData.PortJson)
	if err != nil {
		fmt.Println("Error parsing ports:", err)
		return
	}

	// 输出解析结果
	for _, port := range servicePorts {
		fmt.Printf("Port: %d, Protocol: %s\n", port.Port, port.Protocol)
	}
}
