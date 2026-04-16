// Types matching pkg/cscore JSON responses

export interface TopicSummary {
  name: string;
  category: string;
  title: string;
  description: string;
  has_detail: boolean;
  see_also?: string[];
}

export interface SheetResponse {
  name: string;
  category: string;
  title: string;
  description: string;
  content: string;
  see_also?: string[];
  sections: SectionResponse[];
  has_detail: boolean;
}

export interface SectionResponse {
  title: string;
  level: number;
  content: string;
}

export interface DetailResponse {
  name: string;
  category: string;
  title: string;
  content: string;
  prerequisites?: string[];
  complexity?: string;
}

export interface CategorySummary {
  name: string;
  count: number;
}

export interface CategoryTopicsResponse {
  category: string;
  topics: TopicSummary[];
}

export interface SearchResponse {
  query: string;
  results: SearchResult[];
  count: number;
}

export interface SearchResult {
  topic: string;
  category: string;
  section?: string;
  line?: string;
}

export interface RelatedResponse {
  topic: string;
  related: TopicSummary[];
}

export interface CompareResponse {
  a: CompareSide;
  b: CompareSide;
  all_sections: string[];
  sections_a: string[];
  sections_b: string[];
}

export interface CompareSide {
  name: string;
  category: string;
  description: string;
  sections: number;
  lines: number;
  has_detail: boolean;
  see_also?: string[];
}

export interface LearnPathResponse {
  category: string;
  path: LearnPathEntry[];
}

export interface LearnPathEntry {
  order: number;
  name: string;
  description: string;
  has_detail: boolean;
  prerequisites?: string[];
  prereq_count: number;
}

export interface StatsResponse {
  total_sheets: number;
  detail_pages: number;
  categories: number;
  see_also_coverage: number;
  bookmarks: number;
  total_lines: number;
  per_category: CategorySummary[];
}

export interface CalcResponse {
  expr: string;
  value: number;
  formatted: string;
  hex?: string;
  oct?: string;
  bin?: string;
  unit?: string;
}

export interface SubnetResponse {
  cidr: string;
  network: string;
  broadcast?: string;
  netmask?: string;
  wildcard?: string;
  prefix: number;
  first_host: string;
  last_host: string;
  total_hosts: string;
  usable_hosts: string;
  is_ipv6: boolean;
}

export interface BookmarkToggleResponse {
  topic: string;
  bookmarked: boolean;
}

export interface BookmarkListResponse {
  bookmarks: string[];
}

export interface VerifyResponse {
  topic: string;
  results: VerifyResult[];
  pass: number;
  fail: number;
  total: number;
}

export interface VerifyResult {
  expression: string;
  expected: number;
  got: number;
  pass: boolean;
  line: string;
}

export interface MarkdownResponse {
  html: string;
}

export interface ErrorResponse {
  error: string;
  field?: string;
}
