Feature: Kubernetes 
  In order to test the TrafficTarget
  As a developer
  I need to ensure the specification is accepted by the server

  @k8s
  Scenario: Simple Canary Deployment
    Given the controller is running on Kubernetes