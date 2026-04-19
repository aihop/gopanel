import type { ReqPage } from "@/api/interface"
import type { File } from "@/api/interface/file"
import type { AxiosRequestConfig } from "axios"
import type { ResPage } from "../interface"
import http from "@/api"
import { TimeoutEnum } from "@/enums/http-enum"

export function GetFilesList(params: File.ReqFile) {
	return http.post<File.File>("/file/search", params, TimeoutEnum.T_5M)
}

export function GetUploadList(params: File.SearchUploadInfo) {
	return http.post<ResPage<File.UploadInfo>>("/file/upload/search", params)
}

export function GetFilesTree(params: File.ReqFile) {
	return http.post<File.FileTree[]>("/file/tree", params)
}

export function CreateFile(form: File.FileCreate) {
	return http.post<File.File>("/file/create", form)
}

export function DeleteFile(form: File.FileDelete) {
	return http.post<File.File>("/file/del", form)
}

export function BatchDeleteFile(form: File.FileBatchDelete) {
	return http.post("/file/batch/del", form)
}

export function ChangeFileMode(form: File.FileCreate) {
	return http.post<File.File>("/file/mode", form)
}

export function fileCompressAPI(form: File.FileCompress) {
	return http.post<File.File>("/file/compress", form, TimeoutEnum.T_10M)
}

export function DeCompressFile(form: File.FileDeCompress) {
	return http.post<File.File>("/file/decompress", form, TimeoutEnum.T_10M)
}

export function GetFileContent(params: File.ReqFile) {
	return http.post<File.File>("/file/content", params)
}

export function SaveFileContent(params: File.FileEdit) {
	return http.post<File.File>("/file/save", params)
}

export function CheckFile(path: string) {
	return http.post<boolean>("/file/check", { path })
}

export function BatchCheckFiles(paths: string[]) {
	return http.post<File.ExistFileInfo[]>("/file/batch/check", { paths }, TimeoutEnum.T_5M)
}

export function UploadFileData(params: FormData, config: AxiosRequestConfig) {
	return http.upload<File.File>("/file/upload", params, config)
}

export function ChunkUploadFileData(params: FormData, config: AxiosRequestConfig) {
	return http.upload<File.File>("/file/chunkUpload", params, config)
}

export function RenameRile(params: File.FileRename) {
	return http.post<File.File>("/file/rename", params)
}

export function ChangeOwner(params: File.FileOwner) {
	return http.post<File.File>("/file/owner", params)
}

export function WgetFile(params: File.FileWget) {
	return http.post<File.FileWgetRes>("/file/wget", params)
}

export function MoveFile(params: File.FileMove) {
	return http.post<File.File>("/file/move", params, TimeoutEnum.T_5M)
}

export function DownloadFile(params: File.FileDownload) {
	return http.download<BlobPart>("/file/download", params, { responseType: "blob", timeout: TimeoutEnum.T_40S })
}

export function ComputeDirSize(params: File.DirSizeReq) {
	return http.post<File.DirSizeRes>("/file/size", params, TimeoutEnum.T_5M)
}

export function FileKeys() {
	return http.get<File.FileKeys>("/file/keys")
}

export function getRecycleList(params: ReqPage) {
	return http.post<ResPage<File.RecycleBin>>("/file/recycle/search", params)
}

export function reduceFile(params: File.RecycleBinReduce) {
	return http.post<any>("/file/recycle/reduce", params)
}

export function clearRecycle() {
	return http.post<any>("/file/recycle/clear")
}

export function SearchFavorite(params: ReqPage) {
	return http.post<ResPage<File.Favorite>>("/file/favorite/search", params)
}

export function AddFavorite(path: string) {
	return http.post<any>("/file/favorite", { path })
}

export function ReadByLine(req: File.FileReadByLine) {
	return http.post<any>("/file/read", req)
}

export function DirExist(req: { dir: string }) {
	return http.post<any>("/file/dirExist", req)
}

export function RemoveFavorite(id: number) {
	return http.post<any>("/file/favorite/del", { id })
}

export function BatchChangeRole(params: File.FileRole) {
	return http.post<any>("/file/batch/role", params)
}

export function FileRecycleStatusAPI() {
	return http.post<string>("/file/recycle/status")
}
