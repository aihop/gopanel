package service

// runEnvInjectMarker 在 docker/podman shim 脚本里占位，Sprintf 之后统一替换成
// runEnvInjectSnippet，避免把这段 shell 混进本来就有 %% 转义的格式化字符串里。
const runEnvInjectMarker = "__GOPANEL_RUN_ENV_INJECT__"

// runEnvInjectSnippet 纯脚本流水线里，容器是用户自己的构建脚本起的，面板没法直接写 config.Env。
// 这里借已经挂在 PATH 上的 docker/podman shim，在 run/create 子命令后面补上版本号环境变量，
// 让脚本模式和 Runner 模式一样能在容器里读到版本。
// 注入的 -e 放在最前面，用户脚本里自己写的同名 -e 在后面会覆盖它——用户意图优先。
const runEnvInjectSnippet = `if [ -n "$PIPELINE_VERSION" ]; then
	case "$1" in
		run|create)
			gopanel_sub="$1"; shift
			set -- "$gopanel_sub" -e "GOPANEL_PIPELINE_VERSION=$PIPELINE_VERSION" -e "PIPELINE_VERSION=$PIPELINE_VERSION" "$@"
			;;
		container)
			case "$2" in
				run|create)
					gopanel_sub1="$1"; gopanel_sub2="$2"; shift 2
					set -- "$gopanel_sub1" "$gopanel_sub2" -e "GOPANEL_PIPELINE_VERSION=$PIPELINE_VERSION" -e "PIPELINE_VERSION=$PIPELINE_VERSION" "$@"
					;;
			esac
			;;
	esac
fi`
