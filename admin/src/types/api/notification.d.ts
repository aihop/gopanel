export interface NotificationRecord {
	id: number
	types: string
	userId: number
	toUserId: number
	avatar: string
	title: string
	content: string
	targetId: number
	isRead: number
	status: number
	readAt: number
	createdAt: string
	updatedAt: string
}

export interface Category {
	id: number
	dictId: number
	parentId: number
	slug: string
	name: string
	subtitle: string
	attachId: number
	thumbnail: string
	thumbnailAlt: string
	title: string
	keywords: string
	description: string
	sort: number
	status: number
	template: string
	route: string
	detailTemplate: string
	detailRoute: string
	domain: string
	langs: any
	createdAt: string
	updatedAt: string
}

export interface Article {
	id: number
	slug: string
	title: string
	categoryId: number
	category: Category
	attachId: number
	thumbnail: string
	credits: number
	virtualViews: number
	views: number
	likes: number
	isFavorite: boolean
	favorites: number
	comments: number
	scores: number
	author: string
	source: string
	keywords: string
	description: string
	domain: string
	location: string
	langs: any
	url: string
	tags: any
	status: number
	storeId: number
	createUserId: number
	createdAt: string
	updatedAt: string
}
