@nomad
Feature: Nomad
  In order to test a Canary Deployments on Nomad
  I need to ensure the code funcionality is working as specified

  @nomad_canary_existing
  Scenario: Canary Deployment existing candidate succeeds
    Given the controller is running on Nomad
