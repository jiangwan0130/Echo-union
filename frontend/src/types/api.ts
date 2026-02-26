// ── 通用响应结构 ──

export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
  details?: string;
}

export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export interface PaginatedData<T> {
  list: T[];
  pagination: Pagination;
}

export type PaginatedResponse<T> = ApiResponse<PaginatedData<T>>;

// ── 分页请求参数 ──

export interface PaginationParams {
  page?: number;
  page_size?: number;
}
