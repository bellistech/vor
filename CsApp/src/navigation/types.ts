export type BrowseStackParams = {
  Categories: undefined;
  TopicList: { category: string };
  Sheet: { topic: string; title: string };
  Detail: { topic: string; title: string };
};

export type SearchStackParams = {
  Search: undefined;
  Sheet: { topic: string; title: string };
  Detail: { topic: string; title: string };
};

export type StarredStackParams = {
  Starred: undefined;
  Sheet: { topic: string; title: string };
  Detail: { topic: string; title: string };
};

export type ToolsStackParams = {
  Tools: undefined;
};

export type MoreStackParams = {
  More: undefined;
  Sheet: { topic: string; title: string };
  Detail: { topic: string; title: string };
};
