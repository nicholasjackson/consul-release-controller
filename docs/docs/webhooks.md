---
sidebar_position: 6
---

# Webhook notification

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
    - name: "discord"
      pluginName: "discord"
      config:
        id: "947009151231496821"
        token: "-OoJOZtJJoAjLBhREuuTtTxlP4q3J21gaeIPk8CPzPWXnSxBj"
    - name: "slack"
      pluginName: "slack"
      config:
        url: "https://hooks.slack.com/services/T9JT4868N/34340Q02/h9N1ry9x29quExF3434f7J"
    - name: "slack_custom"
      pluginName: "slack"
      config:
        url: "https://hooks.slack.com/services/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"
        template: |
          Custom template message: State has changed to "{{ .State }}" for
          the release "{{ .Name }}" in the namespace "{{ .Namespace }}".

          The outcome was "{{ .Outcome }}"
```