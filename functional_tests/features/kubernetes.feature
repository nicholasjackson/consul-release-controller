Feature: Kubernetes 
  In order to test a Canary Deployment on Kubernetes
  I need to ensure the code funcionality is working as specified

  @k8s
  Scenario: Simple Canary Deployment existing candidate
    Given the controller is running on Kubernetes
    And I create a new version of the Kubernetes Deployment "../example/kubernetes/api.yaml"
    Then a Kubernetes deployment called "api-deployment" should not exist
      And eventually a call to the URL "http://localhost:18080" contains the text 
        """
        API V1
        """
    When I create a new Canary "../example/kubernetes/canary/api.json"
      Then a Kubernetes deployment called "api-deployment-primary" should exist
      And a Kubernetes deployment called "api-deployment" should not exist
      And a Consul "service-defaults" called "api" should be created
      And a Consul "service-resolver" called "api" should be created
      And a Consul "service-router" called "api" should be created
      And a Consul "service-splitter" called "api" should be created
    When I create a new version of the Kubernetes Deployment "../example/kubernetes/canary/api.yaml"
      Then a Kubernetes deployment called "api-deployment-primary" should exist
      And a Kubernetes deployment called "api-deployment" should exist
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        API V2
        """
    When I delete the Canary "api"
      Then a Kubernetes deployment called "api-deployment-primary" should not exist
      And a Kubernetes deployment called "api-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text 
        """
        API V2
        """