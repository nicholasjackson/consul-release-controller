@nomad
Feature: Nomad
  In order to test a Canary Deployments on Nomad
  I need to ensure the code funcionality is working as specified

  @nomad_canary_existing
  Scenario: Canary Deployment existing candidate succeeds
    Given the controller is running on Nomad
    When I delete the Nomad job "payments-deployment"
      Then a Nomad job called "payments-deployment" should not exist
      And a Nomad job called "payments-primary" should not exist
    And I create a new version of the Nomad job "./config/nomad/payments.hcl" called "payments-deployment"
    Then a Nomad job called "payments-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V1
        """
    When I create a new Nomad release "./config/nomad/payments_release.json"
      Then a Nomad job called "payments-primary" should exist
      And a Nomad job called "payments-deployment" should not exist
      And a Consul "service-defaults" called "payments" should be created
      And a Consul "service-resolver" called "payments" should be created
      And a Consul "service-splitter" called "payments" should be created
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
    When I create a new version of the Nomad job "./config/nomad/payments_canary.hcl" called "payments-deployment"
      Then a Nomad job called "payments-primary" should exist
      And a Nomad job called "payments-deployment" should exist
      And eventually a call to the URL "https://localhost:9443/v1/releases" contains the text
        """
        "status":"state_idle"
        """
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V2
        """
    When I delete the Nomad release "payments"
      Then a Nomad job called "payments-primary" should not exist
      And a Nomad job called "payments-deployment" should exist
      And eventually a call to the URL "http://localhost:18080" contains the text
        """
        Payments V2
        """
