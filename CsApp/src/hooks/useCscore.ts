import { useState, useEffect, useCallback } from 'react';
import { cscore } from '../core/cscore';
import type {
  TopicSummary,
  SheetResponse,
  DetailResponse,
  SearchResponse,
  CategorySummary,
  CategoryTopicsResponse,
  CalcResponse,
  SubnetResponse,
  StatsResponse,
} from '../core/types';

let initialized = false;

export function useInit() {
  const [ready, setReady] = useState(initialized);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (initialized) return;
    (async () => {
      try {
        await cscore.init();
        const dir = await cscore.getDocumentsDir();
        await cscore.setDataDir(dir);
        initialized = true;
        setReady(true);
      } catch (e: any) {
        setError(e.message);
      }
    })();
  }, []);

  return { ready, error };
}

export function useCategories() {
  const [categories, setCategories] = useState<CategorySummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    cscore.categories().then(cats => {
      setCategories(cats);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  return { categories, loading };
}

export function useCategoryTopics(category: string) {
  const [topics, setTopics] = useState<TopicSummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!category) return;
    setLoading(true);
    cscore.categoryTopics(category).then(resp => {
      setTopics(resp.topics);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, [category]);

  return { topics, loading };
}

export function useSheet(name: string) {
  const [sheet, setSheet] = useState<SheetResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!name) return;
    setLoading(true);
    cscore.getSheet(name).then(s => {
      setSheet(s);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, [name]);

  return { sheet, loading };
}

export function useDetail(name: string) {
  const [detail, setDetail] = useState<DetailResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!name) return;
    setLoading(true);
    cscore.getDetail(name).then(d => {
      setDetail(d);
      setLoading(false);
    }).catch(() => {
      setDetail(null);
      setLoading(false);
    });
  }, [name]);

  return { detail, loading };
}

export function useSearch() {
  const [results, setResults] = useState<SearchResponse | null>(null);
  const [loading, setLoading] = useState(false);

  const search = useCallback(async (query: string) => {
    if (!query) {
      setResults(null);
      return;
    }
    setLoading(true);
    try {
      const resp = await cscore.search(query);
      setResults(resp);
    } catch {
      setResults(null);
    }
    setLoading(false);
  }, []);

  return { results, loading, search };
}

export function useCalc() {
  const [result, setResult] = useState<CalcResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const evaluate = useCallback(async (expr: string) => {
    if (!expr) {
      setResult(null);
      setError(null);
      return;
    }
    try {
      const resp = await cscore.calcEval(expr);
      setResult(resp);
      setError(null);
    } catch (e: any) {
      setResult(null);
      setError(e.message);
    }
  }, []);

  return { result, error, evaluate };
}

export function useSubnet() {
  const [result, setResult] = useState<SubnetResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const calculate = useCallback(async (input: string) => {
    if (!input) {
      setResult(null);
      setError(null);
      return;
    }
    try {
      const resp = await cscore.subnetCalc(input);
      setResult(resp);
      setError(null);
    } catch (e: any) {
      setResult(null);
      setError(e.message);
    }
  }, []);

  return { result, error, calculate };
}

export function useStats() {
  const [stats, setStats] = useState<StatsResponse | null>(null);

  useEffect(() => {
    cscore.stats().then(setStats).catch(() => {});
  }, []);

  return stats;
}

export function useMarkdown(content: string) {
  const [html, setHtml] = useState<string>('');

  useEffect(() => {
    if (!content) return;
    cscore.renderMarkdown(content).then(resp => {
      setHtml(resp.html);
    }).catch(() => {});
  }, [content]);

  return html;
}
