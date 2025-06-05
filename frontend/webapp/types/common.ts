export interface PaginatedData<T = unknown> {
  nextPage: string
  items: T[]
}
