import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useReducer,
} from 'react';
import { cscore } from './cscore';

interface BookmarkContextValue {
  starred: Set<string>;
  toggle: (topic: string) => Promise<void>;
  isStarred: (topic: string) => boolean;
}

const BookmarkContext = createContext<BookmarkContextValue>({
  starred: new Set(),
  toggle: async () => {},
  isStarred: () => false,
});

type Action =
  | { type: 'INIT'; bookmarks: string[] }
  | { type: 'TOGGLE'; topic: string; bookmarked: boolean };

function reducer(state: Set<string>, action: Action): Set<string> {
  switch (action.type) {
    case 'INIT':
      return new Set(action.bookmarks);
    case 'TOGGLE': {
      const next = new Set(state);
      if (action.bookmarked) {
        next.add(action.topic);
      } else {
        next.delete(action.topic);
      }
      return next;
    }
    default:
      return state;
  }
}

export function BookmarkProvider({ children }: { children: React.ReactNode }) {
  const [starred, dispatch] = useReducer(reducer, new Set<string>());

  useEffect(() => {
    cscore
      .bookmarkList()
      .then(resp => dispatch({ type: 'INIT', bookmarks: resp.bookmarks }))
      .catch(() => {});
  }, []);

  const toggle = useCallback(async (topic: string) => {
    try {
      const resp = await cscore.bookmarkToggle(topic);
      dispatch({ type: 'TOGGLE', topic, bookmarked: resp.bookmarked });
    } catch {}
  }, []);

  const isStarred = useCallback((topic: string) => starred.has(topic), [starred]);

  return (
    <BookmarkContext.Provider value={{ starred, toggle, isStarred }}>
      {children}
    </BookmarkContext.Provider>
  );
}

export function useBookmarks() {
  return useContext(BookmarkContext);
}
