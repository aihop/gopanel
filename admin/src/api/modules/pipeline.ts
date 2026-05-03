import http from "@/api";
import { ResPage } from "@/api/interface/index";
import { Pipeline } from "../interface/pipeline";

export const getPipelinePage = (params: { page: number; limit: number }) => {
  return http.get<ResPage<Pipeline.ResPipeline>>(`/pipeline/list`, params);
};

export const createPipeline = (params: Pipeline.ReqCreate) => {
  return http.post(`/pipeline`, params);
};

export const updatePipeline = (params: Pipeline.ReqUpdate) => {
  return http.put(`/pipeline`, params);
};

export const deletePipeline = (params: { id: number }) => {
  return http.delete(`/pipeline`, params);
};

export const runPipeline = (params: { id: number; version: string }) => {
  return http.post<{ recordId: number }>(`/pipeline/run`, params);
};

export const stopPipeline = (params: { id: number }) => {
  return http.post(`/pipeline/stop`, params);
};

export const getPipelineRecords = (params: { pipelineId: number; page: number; limit: number }) => {
  return http.get<ResPage<Pipeline.ResRecord>>(`/pipeline/records`, params);
};

export const getPipelineReleases = (params: { pipelineId: number; page: number; limit: number }) => {
  return http.get<ResPage<Pipeline.ResRelease>>(`/pipeline/releases`, params);
};

export const getPipelineRelease = (params: { id: number }) => {
  return http.get<Pipeline.ResRelease>(`/pipeline/release`, params);
};

export const publishPipelineRelease = (params: { id: number }) => {
  return http.post<Pipeline.ResRelease>(`/pipeline/release/publish`, params);
};

export const deletePipelineRecord = (params: { id: number }) => {
  return http.delete(`/pipeline/record`, params);
};

export const detectPipelineRunnerPreset = (params: Pipeline.ReqDetectRunnerPreset) => {
  return http.post<Pipeline.ResDetectRunnerPreset>(`/pipeline/detect`, params);
};
