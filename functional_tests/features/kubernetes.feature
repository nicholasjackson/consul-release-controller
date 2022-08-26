@k8s
Feature: Kubernetes
  In order to test a Canary Deployments on Kubernetes
  I need to ensure the code funcionality is working as specified

  @k8s_canary_existing
  Scenario: Canary Deployment existing candidate succeeds
    Given the controller is running on Kubernetes
    When I delete the Kubernetes deployment "payments-deployment"
      Then a Kubernetes deployment called "payments-deployment" should not exist
      And a Kubernetes deployment called "payments-primary" should not exist
    And I create a new version of the Kubernetes deployment "./config/kubernetes/payments.yaml"
    Then a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V1
        """
    When I create a new Kubernetes release "./config/kubernetes/payments_release.yaml"
      Then a Kubernetes deployment called "payments-primary" should exist
      And a Kubernetes deployment called "payments-deployment" should not exist
      And a Consul "service-defaults" called "payments" should be created
      And a Consul "service-resolver" called "payments" should be created
      And a Consul "service-splitter" called "payments" should be created
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
    When I create a new version of the Kubernetes deployment "./config/kubernetes/payments_canary.yaml"
      Then a Kubernetes deployment called "payments-primary" should exist
      And a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V2
        """
    When I delete the Kubernetes release "payments"
      Then a Kubernetes deployment called "payments-primary" should not exist
      And a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V2
        """

  @k8s_canary_none
  Scenario: Canary Deployment no candidate succeeds
    Given the controller is running on Kubernetes
    When I delete the Kubernetes deployment "payments-deployment"
      Then a Kubernetes deployment called "payments-deployment" should not exist
      And a Kubernetes deployment called "payments-primary" should not exist
    When I create a new Kubernetes release "./config/kubernetes/payments_release.yaml"
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
      And a Consul "service-defaults" called "payments" should be created
      And a Consul "service-resolver" called "payments" should be created
    When I create a new version of the Kubernetes deployment "./config/kubernetes/payments.yaml"
      Then a Kubernetes deployment called "payments-primary" should exist
      And a Kubernetes deployment called "payments-deployment" should not exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V1
        """
      And a Consul "service-splitter" called "payments" should be created
    When I create a new version of the Kubernetes deployment "./config/kubernetes/payments_canary.yaml"
      Then a Kubernetes deployment called "payments-primary" should exist
      And a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V2
        """
    When I delete the Kubernetes release "payments"
      Then a Kubernetes deployment called "payments-primary" should not exist
      And a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V2
        """

  @k8s_canary_rollback
  Scenario: Canary Deployment with candidate rollsback
    Given the controller is running on Kubernetes
    When I delete the Kubernetes deployment "payments-deployment"
    And I create a new version of the Kubernetes deployment "./config/kubernetes/payments.yaml"
    Then a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V1
        """
    And I create a new Kubernetes release "./config/kubernetes/payments_release.yaml"
      Then a Kubernetes deployment called "payments-primary" should exist
      And a Kubernetes deployment called "payments-deployment" should not exist
      And a Consul "service-defaults" called "payments" should be created
      And a Consul "service-resolver" called "payments" should be created
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
    When I create a new version of the Kubernetes deployment "./config/kubernetes/payments_with_error.yaml"
      Then a Kubernetes deployment called "payments-primary" should exist
      And a Kubernetes deployment called "payments-deployment" should exist
      And a Consul "service-splitter" called "payments" should be created
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V1
        """
    When I delete the Kubernetes release "payments"
      Then a Kubernetes deployment called "payments-primary" should not exist
      And a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V1
        """

  @k8s_canary_with_post_deployment_test
  Scenario: Canary Deployment with passing post deployment test succeeds
    Given the controller is running on Kubernetes
    When I delete the Kubernetes deployment "payments-deployment"
      Then a Kubernetes deployment called "payments-deployment" should not exist
      And a Kubernetes deployment called "payments-primary" should not exist
    When I create a new Kubernetes release "./config/kubernetes/payments_release_with_check.yaml"
      Then a Consul "service-defaults" called "payments" should be created
      And a Consul "service-resolver" called "payments" should be created
      And a Consul "service-router" called "consul-release-controller-upstreams" should be created
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
    When I create a new version of the Kubernetes deployment "./config/kubernetes/payments.yaml"
      Then a Kubernetes deployment called "payments-primary" should exist
      Then a Kubernetes deployment called "payments-deployment" should not exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V1
        """
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
    When I create a new version of the Kubernetes deployment "./config/kubernetes/payments_canary.yaml"
      Then a Kubernetes deployment called "payments-primary" should exist
      And a Kubernetes deployment called "payments-deployment" should exist
      And a Consul "service-splitter" called "payments" should be created
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V2
        """
    When I delete the Kubernetes release "payments"
      Then a Kubernetes deployment called "payments-primary" should not exist
      And a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V2
        """

  @k8s_canary_with_post_deployment_test_fail
  Scenario: Canary Deployment with failing post deployment test rollsback
    Given the controller is running on Kubernetes
    When I delete the Kubernetes deployment "payments-deployment"
    And I create a new version of the Kubernetes deployment "./config/kubernetes/payments.yaml"
    Then a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V1
        """
    And I create a new Kubernetes release "./config/kubernetes/payments_release_with_check.yaml"
      Then a Kubernetes deployment called "payments-primary" should exist
      And a Kubernetes deployment called "payments-deployment" should not exist
      And a Consul "service-defaults" called "payments" should be created
      And a Consul "service-resolver" called "payments" should be created
      And a Consul "service-router" called "consul-release-controller-upstreams" should be created
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
    When I create a new version of the Kubernetes deployment "./config/kubernetes/payments_with_error.yaml"
      Then a Kubernetes deployment called "payments-primary" should exist
      And a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V1
        """
    When I delete the Kubernetes release "payments"
      Then a Kubernetes deployment called "payments-primary" should not exist
      And a Kubernetes deployment called "payments-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V1
        """
