export interface Vectormap {
	canvas: HTMLCanvasElement
	container: HTMLBaseElement
	linesGroup: {
		canvas: HTMLCanvasElement
		node: HTMLBaseElement
	}
	params: Params
	regions: { [key: string]: Region }
	scale: number
	transX: number
	transY: number
	_baseScale: number
	_baseTransX: number
	_baseTransY: number
	_defaultHeight: number
	_defaultWidth: number
	_height: number
	_width: number
	_lines: { [key: string]: { config: { from: string; to: string }; shape: Shape } }
	_mapData: MapData
	_markerLabelsGroup: {
		canvas: HTMLCanvasElement
		node: HTMLBaseElement
	}
	_markers: Marker[]
	_markersGroup: {
		canvas: HTMLCanvasElement
		node: HTMLBaseElement
	}
	_regionLabelsGroup: {
		canvas: HTMLCanvasElement
		node: HTMLBaseElement
	}
	updateSize: () => void
}

export interface Params {
	map: string
	backgroundColor: string
	draggable: boolean
	zoomButtons: boolean
	zoomOnScroll: boolean
	zoomOnScrollSpeed: number
	zoomMax: number
	zoomMin: number
	zoomAnimate: boolean
	showTooltip: boolean
	zoomStep: number
	bindTouchEvents: boolean
	lineStyle: LineStyle
	markersSelectable: boolean
	markersSelectableOne: boolean
	markerStyle: Style
	markerLabelStyle: Style
	regionsSelectable: boolean
	regionsSelectableOne: boolean
	regionStyle: Style
	regionLabelStyle: Style
	selector: string
	markers: Marker[]
	lines: Line[]
	labels: Labels
}

export interface Labels {
	markers: Marker[]
}

export interface LineStyle {
	stroke: string
	strokeWidth: number
	strokeLinecap: string
	strokeDasharray: string
	animation: boolean
}

export interface Line {
	from: string
	to: string
}

interface Region {
	config: {
		path: string
		name: string
	}
	element: {
		shape: Shape
		map: Vectormap
	}
}

interface Shape {
	isHovered: boolean
	isSelected: boolean
	node: HTMLBaseElement
	style: Style
}

interface Style {
	initial: StyleParams
	hover: StyleParams
	selected: StyleParams
	selectedHover: StyleParams
	current: StyleParams
}

interface StyleParams {
	fill?: string | number
	fillOpacity?: string | number
	stroke?: string | number
	strokeWidth?: string | number
	cursor?: string | number
	backgroundColor?: string | number
	fontFamily?: string
	fontSize?: number
	fontWeight?: number
}

export interface MapData {
	insets: Inset[]
	paths: { [key: string]: Path }
	height: number
	projection: Projection
	width: number
}

export interface Inset {
	width: number
	top: number
	height: number
	bbox: Bbox[]
	left: number
}

export interface Bbox {
	y: number
	x: number
}

export interface Path {
	path: string
	name: string
}

export interface Projection {
	type: string
	centralMeridian: number
}

interface Marker {
	name: string
	config: {
		name: string
		coords: number[]
	}
	element: MapElement
	_uid: string
}

interface MapElement {
	config: {
		cx: number
		cy: number
		group: {
			canvas: HTMLCanvasElement
			node: HTMLBaseElement
		}
		index: string | number
		isRecentlyCreated: boolean
		label: { render: (marker: Marker) => string }
		labelsGroup: {
			canvas: HTMLCanvasElement
			node: HTMLBaseElement
		}
		map: Vectormap
		marker: Marker
		style: Style
	}
	label: Shape
	shape: Shape
	_isImage: boolean
	_labelX: number
	_labelY: number
	map: Vectormap
	offset: [number, number]
}

interface Tooltip {
	_map: Vectormap
	_tooltip: HTMLBaseElement
	css: (style: StyleParams) => void
}
