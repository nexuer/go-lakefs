import { api } from "./client"

export interface Repo {
    id: string,
    creation_date: number
    storage_namespace: string
}

export interface ListReposRequest {
    search: string
}

export function fetchListRepos(params: ListReposRequest): Promise<Repo[]> {
    return api("/api/v1/repos", {
        method: "GET",
        query: params,
    })
}

export interface CreateRepoRequest {
    name: string
}

export function fetchCreateRepo(params: CreateRepoRequest): Promise<Repo> {
    return api("/api/v1/repos", {
        method: "POST",
        body: params,
    })
}

export function fetchDeleteRepo(id: string): Promise<Repo> {
    return api(`/api/v1/repos/${id}`, {
        method: "DELETE",
    })
}

export interface StagingLocation {
    physical_address: string
    presigned_url: string
    presigned_url_expiry: number
}

export function fetchFilePresignUrl(id: string, filename: string): Promise<StagingLocation> {
    return api(`/api/v1/repos/${id}/branches/main/presign/${filename}`, {
        method: "GET",
    })
}

export interface StagingMetadata {
    staging: StagingLocation
    checksum: string
    size_bytes: number
    user_metadata: Record<string, string>
    content_type: string
    mtime: number
    force: boolean
}

export interface CreateFilesRequest {
    files: StagingMetadata[]
}

export interface CreateFilesResponse {
    total: number
}

export function fetchCreateFiles(id: string, body: CreateFilesRequest): Promise<CreateFilesResponse> {
    return api(`/api/v1/repos/${id}/branches/main/files`, {
        method: "POST",
        body: body,
    })
}

export interface Object {
    path: string
    checksum: string
    size_bytes: number
    mtime: number
    content_type: string
}

export function fetchListObjects(id: string): Promise<Object[]> {
    return api(`/api/v1/repos/${id}/branches/main/files`, {
        method: "GET",
    })
}

export function fetchDeleteObject(id: string, filename: string): Promise<CreateFilesResponse> {
    return api(`/api/v1/repos/${id}/branches/main/files/${filename}`, {
        method: "DELETE",
    })
}

export function fetchDownloadFile(id: string, filename: string) {
    return api(`/api/v1/repos/${id}/branches/main/files/${filename}`, {
        method: "GET",
        responseType: "blob"
    })
}

export function fetchFilePreview(id: string, filename: string) {
    return api(`/api/v1/repos/${id}/branches/main/files/${filename}?preview=true`, {
        method: "GET",
        responseType: "stream",
    })
}