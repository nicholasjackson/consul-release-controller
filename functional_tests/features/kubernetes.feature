@k8s
Feature: Kubernetes 
  In order to test a Canary Deployments on Kubernetes
  I need to ensure the code funcionality is working as specified

  @canary_existing
  Scenario: Canary Deployment existing candidate
    Given the controller is running on Kubernetes
    And I create a new version of the Kubernetes deployment "../example/kubernetes/api.yaml"
    Then a Kubernetes deployment called "api-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text 
        """
        API V1
        """
    When I create a new Kubernetes release "../example/kubernetes/canary/api_release.yaml"
      Then a Kubernetes deployment called "api-deployment-primary" should exist
      And a Kubernetes deployment called "api-deployment" should not exist
      And a Consul "service-defaults" called "api" should be created
      And a Consul "service-resolver" called "api" should be created
      And a Consul "service-router" called "api" should be created
      And a Consul "service-splitter" called "api" should be created
    When I create a new version of the Kubernetes deployment "../example/kubernetes/canary/api.yaml"
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
    When I delete the Kubernetes release "api"
      Then a Kubernetes deployment called "api-deployment-primary" should not exist
      And a Kubernetes deployment called "api-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text 
        """
        API V2
        """

  @canary_none
  Scenario: Canary Deployment no candidate
    Given the controller is running on Kubernetes
    When I delete the Kubernetes deployment "api-deployment"
    And I create a new Kubernetes release "../example/kubernetes/canary/api_release.yaml"
      Then a Kubernetes deployment called "api-deployment-primary" should not exist
      And a Kubernetes deployment called "api-deployment" should not exist
    When I create a new version of the Kubernetes deployment "../example/kubernetes/api.yaml"
      Then a Kubernetes deployment called "api-deployment-primary" should exist
      Then a Kubernetes deployment called "api-deployment" should not exist
      And eventually a call to the URL "http://localhost:18080" contains the text 
        """
        API V1
        """
      And a Consul "service-defaults" called "api" should be created
      And a Consul "service-resolver" called "api" should be created
      And a Consul "service-router" called "api" should be created
      And a Consul "service-splitter" called "api" should be created
    When I create a new version of the Kubernetes deployment "../example/kubernetes/canary/api.yaml"
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
    When I delete the Kubernetes release "api"
      Then a Kubernetes deployment called "api-deployment-primary" should not exist
      And a Kubernetes deployment called "api-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text 
        """
        API V2
        """
  
  @canary_rollback
  Scenario: Canary Deployment with Rollback
    Given the controller is running on Kubernetes
    When I delete the Kubernetes deployment "api-deployment"
    And I create a new version of the Kubernetes deployment "../example/kubernetes/api.yaml"
    Then a Kubernetes deployment called "api-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text 
        """
        API V1
        """
    And I create a new Kubernetes release "../example/kubernetes/canary/api_release.yaml"
      Then a Kubernetes deployment called "api-deployment-primary" should exist
      And a Kubernetes deployment called "api-deployment" should not exist
      And a Consul "service-defaults" called "api" should be created
      And a Consul "service-resolver" called "api" should be created
      And a Consul "service-router" called "api" should be created
      And a Consul "service-splitter" called "api" should be created
    When I create a new version of the Kubernetes deployment "../example/kubernetes/canary/api_with_error.yaml"
      Then a Kubernetes deployment called "api-deployment-primary" should exist
      And a Kubernetes deployment called "api-deployment" should exist
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        API V1
        """
    When I delete the Kubernetes release "api"
      Then a Kubernetes deployment called "api-deployment-primary" should not exist
      And a Kubernetes deployment called "api-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text 
        """
        API V1
        """