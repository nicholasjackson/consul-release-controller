// allow nomad-pack to set the job name

[[- define "job_name" -]]
[[- if eq .consul_release_controller.job_name "" -]]
[[- .nomad_pack.pack.name | quote -]]
[[- else -]]
[[- .consul_release_controller.job_name | quote -]]
[[- end -]]
[[- end -]]

// only deploys to a region if specified

[[- define "region" -]]
[[- if not (eq .consul_release_controller.region "") -]]
region = [[ .consul_release_controller.region | quote]]
[[- end -]]
[[- end -]]
