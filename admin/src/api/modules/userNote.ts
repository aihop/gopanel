import http from "@/api"

export interface UserNote {
	id: number
	userID: number
	content: string
	createdAt: string
	updatedAt: string
}

export const userNoteGetAPI = () => http.get<UserNote>("/user/note")

export const userNoteSaveAPI = (content: string) => http.post<UserNote>("/user/note", { content })
