module gitlab.com/zynero/shared/app/example

go 1.24.2

require (
	gitlab.com/zynero/shared/app v0.1.0
	gitlab.com/zynero/shared/cache v0.1.5
	gitlab.com/zynero/shared/database v0.1.5
	gitlab.com/zynero/shared/healthcheck v0.1.6
	gitlab.com/zynero/shared/logger v0.1.10
	gitlab.com/zynero/shared/metrics v0.1.6
	gitlab.com/zynero/shared/server v0.1.5
	gitlab.com/zynero/shared/transport v0.1.5
)

replace gitlab.com/zynero/shared/app => ../
