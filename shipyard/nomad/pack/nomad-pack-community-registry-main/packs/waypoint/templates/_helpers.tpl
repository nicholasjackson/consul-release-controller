// allow nomad-pack to set the job name

[[- define "job_name" -]]
[[- if eq .waypoint.job_name "" -]]
[[- .nomad_pack.pack.name | quote -]]
[[- else -]]
[[- .waypoint.job_name | quote -]]
[[- end -]]
[[- end -]]

// only deploys to a region if specified

[[- define "region" -]]
[[- if not (eq .waypoint.region "") -]]
region = [[ .waypoint.region | quote]]
[[- end -]]
[[- end -]]
