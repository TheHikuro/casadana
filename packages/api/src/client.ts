import Axios, { AxiosError, AxiosRequestConfig } from "axios"

export const AXIOS_INSTANCE = Axios.create({
  baseURL:
    (typeof import.meta !== "undefined" && (import.meta as any).env?.VITE_API_BASE_URL) || "",
  headers: { "Content-Type": "application/json" },
  withCredentials: true,
})

export type ApiErrorBody = { error: { code: string; message: string } }

export class ApiError extends Error {
  public code: string
  public status: number
  constructor(code: string, status: number, message: string) {
    super(message)
    this.name = "ApiError"
    this.code = code
    this.status = status
  }
  static fromAxios(e: AxiosError<ApiErrorBody>): ApiError {
    const status = e.response?.status ?? 0
    const body = e.response?.data?.error
    return new ApiError(body?.code ?? "NETWORK", status, body?.message ?? e.message)
  }
}

export const customAxios = <T>(config: AxiosRequestConfig): Promise<T> => {
  return AXIOS_INSTANCE({ ...config })
    .then((res) => res.data as T)
    .catch((e: AxiosError<ApiErrorBody>) => {
      throw ApiError.fromAxios(e)
    })
}
