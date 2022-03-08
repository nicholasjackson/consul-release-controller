---
sidebar_position: 6
---

# Webhook notification

Consul Release Controller supports Webhooks for notifications, currently `Discord` and `Slack` are supported with default
and custom messages. 

**States**

Webhooks are called when Consul Release Controller enters the following states:

| State           | Results                                    | Description                         |
| --------------- | ------------------------------------------ | ----------------------------------- |
| state_configure | event_fail, event_configured               | Fired when a new release is created |
| state_deploy    | event_fail, event_complete                 | Fired when a new deployment is created |
| state_monitor   | event_fail, event_unhealthy, event_healthy | Fired when monitoring a deployment |
| state_scale     | event_fail, event_scaled                   | Fired when scaling a deployment |
| state_promote   | event_fail, event_promoted                 | Fired when promoting a candidate to the primary |
| state_rollback  | event_fail, event_complete                 | Fired when rolling back a failed deployment |
| state_destroy   | event_fail, event_complete                 | Fired when removing a previously configured release |

These states can be used to filter webhooks using the `status` parameter to reduce ChatOps noise.

## Slack Webhooks

The following example shows how to configure a webhook that can post to Slack channels.

```yaml
  webhooks:
    - name: "slack"
      pluginName: "slack"
      config:
        url: "https://hooks.slack.com/services/T9JT4868N/34340Q02/h9N1ry9x29quExF3434f7J"
    - name: "slack_custom"
      pluginName: "slack"
      config:
        url: "https://hooks.slack.com/services/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"
        status:
          - state_deploy
          - state_scale
        template: |
          Custom template message: State has changed to "{{ .State }}" for
          the release "{{ .Name }}" in the namespace "{{ .Namespace }}".

          The outcome was "{{ .Outcome }}"
```

### Parameters

| Name     | Type     | Required | Description           |
| -------- | -------- | -------- | --------------------- |
| url      | string   | Yes      | The Slack Webhook URL |
| template | string   | No       | Optional template to replace default Webhook message |
| status   | []string | No       | List of statuses to send Webhook message, omitting this parameter calls the webhook for all statuses | 

## Discord Webhooks

The following example shows how to configure

```yaml
  webhooks:
    - name: "discord_custom"
      pluginName: "discord"
      config:
        id: "94700915179898981"
        token: "-OoJOZtJJoAjLBhREuuTtTxlP4q3J219SOGIF5X4O1rro34344wdfwfIPk8CPzPWXnSxBj"
        template: |
          Custom template message: State has changed to "{{ .State }}" for
          the release "{{ .Name }}" in the namespace "{{ .Namespace }}".

          The outcome was "{{ .Outcome }}"
        status:
          - state_deploy
          - state_scale
    - name: "discord"
      pluginName: "discord"
      config:
        id: "947009151231496821"
        token: "-OoJOZtJJoAjLBhREuuTtTxlP4q3J21gaeIPk8CPzPWXnSxBj"
```

### Parameters

| Name     | Type     | Required | Description           |
| -------- | -------- | -------- | --------------------- |
| id       | string   | Yes      | The Discord Webhook ID |
| token    | string   | Yes      | The Discord Webhook token |
| template | string   | No       | Optional template to replace default Webhook message |
| status   | []string | No       | List of statuses to send Webhook message, omitting this parameter calls the webhook for all statuses | 

## Custom Messages

Rather than have the Webhook send the default messages you can configure a template to be used instead.

Templates are written using Go Template, you can reference the Template Variables or use any of the flow control and
default functions.

```go
Consul Release Controller state has changed to "{{ .State }}" for
the release "{{ .Name }}" in the namespace "{{ .Namespace }}".

Primary traffic: {{ .PrimaryTraffic }}
Candidate traffic: {{ .CandidateTraffic }}

{{ if ne .Error "" }}
An error occurred when processing: {{ .Error }}
{{ else }}
The outcome is "{{ .Outcome }}"
{{ end }}
```

### Template Variables

| Name             | Type   | Description           |
| ---------------- | ------ | --------------------- |
| Title            | string | The Title for the Webhook message |
| Name             | string | The Name of the release |
| State            | string | Current state of the release |
| Outcome          | string | The outcome of the status, success, fail, etc. See States table above | 
| PrimaryTraffic   | int    | Percentage of Traffic distributed to the Primary instance 0-100 | 
| CandidateTraffic | int    | Percentage of Traffic distributed to the Candidate instance 0-100 | 
| Error            | string | An error message if the status failed | 