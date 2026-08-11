## What changed / 改了什么

<!-- One or two sentences. Link the issue if there is one. -->

## Why / 为什么

<!-- The constraint or bug that made this necessary. -->

## How it was verified / 怎么验证的

<!--
Name the checks you actually ran, not the ones that exist.
A regression test that fails without the fix is the strongest evidence.
写实际跑过的检查。「不打补丁就失败的回归测试」是最有力的证据。
-->

- [ ] `go test ./... -count=1`
- [ ] `npm run test:unit && npm run type-check && npm run lint` (in `admin/`)
- [ ] `bash scripts/check-file-size.sh`
- [ ] Manually exercised in a running panel / 在运行中的面板里实际操作过

## Notes for reviewers / 给评审的说明

<!-- Anything deliberately left out, or a trade-off worth arguing about. -->
