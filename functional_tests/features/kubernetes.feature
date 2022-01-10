Feature: Kubernetes 
  In order to test a Canary Deployment on Kubernetes
  I need to ensure the code funcionality is working

  @k8s
  Scenario: Simple Canary Deployment
    Given the controller is running on Kubernetes
    And a call to the URL "http://localhost:18080" contains the text "API V1"
    When I create a new Canary "../example/kubernetes/canary/api.json"
    And I create a new version of the Kubernetes Deployment "../example/kubernetes/canary/api.yaml"
    Then a Kubernetes deployment called "api-deployment-primary" should be created
    And a Consul "service-defaults" called "api" should be created
    And a Consul "service-resolver" called "api" should be created
    And a Consul "service-splitter" called "api" should be created
    And a Consul "service-router" called "api" should be created