// Typed TypeScript API for the Go core via native bridge.
// All calls are async (dispatched to background thread in Swift).
import { NativeModules } from 'react-native';
import type {
  TopicSummary,
  SheetResponse,
  DetailResponse,
  SearchResponse,
  CategorySummary,
  CategoryTopicsResponse,
  RelatedResponse,
  CompareResponse,
  LearnPathResponse,
  StatsResponse,
  CalcResponse,
  SubnetResponse,
  BookmarkToggleResponse,
  BookmarkListResponse,
  VerifyResponse,
  MarkdownResponse,
  ErrorResponse,
} from './types';

const { CscoreModule } = NativeModules;

function parseJSON<T>(json: string): T | ErrorResponse {
  try {
    return JSON.parse(json) as T;
  } catch {
    return { error: 'Invalid JSON from Go core' };
  }
}

function isError(v: unknown): v is ErrorResponse {
  return typeof v === 'object' && v !== null && 'error' in v;
}

export const cscore = {
  async init(): Promise<void> {
    await CscoreModule.init();
  },

  async setDataDir(path: string): Promise<void> {
    await CscoreModule.setDataDir(path);
  },

  async listTopics(): Promise<TopicSummary[]> {
    const json = await CscoreModule.listTopicsJSON();
    const result = parseJSON<TopicSummary[]>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async getSheet(name: string): Promise<SheetResponse> {
    const json = await CscoreModule.getSheetJSON(name);
    const result = parseJSON<SheetResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async getDetail(name: string): Promise<DetailResponse> {
    const json = await CscoreModule.getDetailJSON(name);
    const result = parseJSON<DetailResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async randomTopic(): Promise<SheetResponse> {
    const json = await CscoreModule.randomTopicJSON();
    const result = parseJSON<SheetResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async search(query: string): Promise<SearchResponse> {
    const json = await CscoreModule.searchJSON(query);
    const result = parseJSON<SearchResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async categories(): Promise<CategorySummary[]> {
    const json = await CscoreModule.categoriesJSON();
    const result = parseJSON<CategorySummary[]>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async categoryTopics(category: string): Promise<CategoryTopicsResponse> {
    const json = await CscoreModule.categoryTopicsJSON(category);
    const result = parseJSON<CategoryTopicsResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async related(name: string): Promise<RelatedResponse> {
    const json = await CscoreModule.relatedJSON(name);
    const result = parseJSON<RelatedResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async compare(a: string, b: string): Promise<CompareResponse> {
    const json = await CscoreModule.compareJSON(a, b);
    const result = parseJSON<CompareResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async learnPath(category: string): Promise<LearnPathResponse> {
    const json = await CscoreModule.learnPathJSON(category);
    const result = parseJSON<LearnPathResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async stats(): Promise<StatsResponse> {
    const json = await CscoreModule.statsJSON();
    const result = parseJSON<StatsResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async calcEval(expr: string): Promise<CalcResponse> {
    const json = await CscoreModule.calcEval(expr);
    const result = parseJSON<CalcResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async subnetCalc(input: string): Promise<SubnetResponse> {
    const json = await CscoreModule.subnetCalc(input);
    const result = parseJSON<SubnetResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async bookmarkToggle(topic: string): Promise<BookmarkToggleResponse> {
    const json = await CscoreModule.bookmarkToggle(topic);
    const result = parseJSON<BookmarkToggleResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async bookmarkList(): Promise<BookmarkListResponse> {
    const json = await CscoreModule.bookmarkList();
    const result = parseJSON<BookmarkListResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async bookmarkIsStarred(topic: string): Promise<boolean> {
    return CscoreModule.bookmarkIsStarred(topic);
  },

  async verify(topic: string): Promise<VerifyResponse> {
    const json = await CscoreModule.verifyJSON(topic);
    const result = parseJSON<VerifyResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async renderMarkdown(md: string): Promise<MarkdownResponse> {
    const json = await CscoreModule.renderMarkdownToHTML(md);
    const result = parseJSON<MarkdownResponse | ErrorResponse>(json);
    if (isError(result)) throw new Error(result.error);
    return result;
  },

  async getDocumentsDir(): Promise<string> {
    return CscoreModule.getDocumentsDir();
  },
};
