import { describe, expect, it } from "vitest"
import { moveCodeProject, reconcileCodeProjectOrder, sortCodeProjectsByOrder } from "./codeProjectOrder"

describe("reconcileCodeProjectOrder", () => {
	it("保留已有顺序，新项目置顶，已删除项目移除", () => {
		expect(reconcileCodeProjectOrder([4, 3, 2], [2, 1, 3])).toEqual([4, 2, 3])
	})

	it("首次使用时保持接口原始顺序", () => {
		expect(reconcileCodeProjectOrder([3, 2, 1], [])).toEqual([3, 2, 1])
	})
})

describe("moveCodeProject", () => {
	it("支持拖到目标前后", () => {
		expect(moveCodeProject([1, 2, 3, 4], 1, 3, "after")).toEqual([2, 3, 1, 4])
		expect(moveCodeProject([1, 2, 3, 4], 4, 2, "before")).toEqual([1, 4, 2, 3])
	})

	it("无效目标保持原引用", () => {
		const order = [1, 2, 3]
		expect(moveCodeProject(order, 1, 1, "before")).toBe(order)
		expect(moveCodeProject(order, 9, 1, "before")).toBe(order)
	})
})

describe("sortCodeProjectsByOrder", () => {
	it("按偏好排序且不修改输入数组", () => {
		const projects = [{ id: 1 }, { id: 2 }, { id: 3 }]
		expect(sortCodeProjectsByOrder(projects, [3, 1, 2]).map(project => project.id)).toEqual([3, 1, 2])
		expect(projects.map(project => project.id)).toEqual([1, 2, 3])
	})
})
