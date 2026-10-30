import type {
  HydroGoClient,
  ContentQuery,
  ContentList,
  ContentSubscribeOptions,
  Page,
  NewsItem,
  NewsPage,
} from "./hydrogo.js";

export type QueryClientLike = {
  invalidateQueries(filters: { queryKey: readonly unknown[] }): Promise<void> | void;
  setQueryData?(queryKey: readonly unknown[], updater: unknown): void;
};

export function collectionQueryOptions<T = Record<string, unknown>>(
  client: HydroGoClient,
  type: string,
  query?: ContentQuery
) {
  return {
    queryKey: ["hydrogo", "content", type, "list", query ?? {}] as const,
    queryFn: (): Promise<ContentList<T>> => client.listContent<T>(type, query),
  };
}

export function collectionRecordQueryOptions<T = Record<string, unknown>>(
  client: HydroGoClient,
  type: string,
  id: string,
  locale?: string
) {
  return {
    queryKey: ["hydrogo", "content", type, "record", id, locale ?? ""] as const,
    queryFn: (): Promise<T> => client.getContent<T>(type, id, locale),
  };
}

export const createCollectionQuery = collectionQueryOptions;
export const createCollectionRecordQuery = collectionRecordQueryOptions;

export type CollectionMutationVariables<T = Record<string, unknown>> = {
  action: "create" | "update" | "delete";
  id?: string;
  data?: Partial<T>;
  token?: string;
};

export function createCollectionMutation<T = Record<string, unknown>>(
  client: HydroGoClient,
  type: string,
  queryClient?: QueryClientLike
) {
  return {
    mutationFn: async (variables: CollectionMutationVariables<T>): Promise<T | void> => {
      switch (variables.action) {
        case "create":
          return client.createContent<T>(type, variables.data ?? {}, variables.token);
        case "update":
          if (!variables.id) throw new Error("Record id is required for update action");
          return client.updateContent<T>(type, variables.id, variables.data ?? {}, variables.token);
        case "delete":
          if (!variables.id) throw new Error("Record id is required for delete action");
          return client.deleteContent(type, variables.id, variables.token);
        default:
          throw new Error(`Unsupported mutation action: ${(variables as { action: string }).action}`);
      }
    },
    onSuccess: async () => {
      if (queryClient) {
        await queryClient.invalidateQueries({ queryKey: ["hydrogo", "content", type] });
      }
    },
  };
}

export function autoSyncRealtimeCollection<T = Record<string, unknown>>(
  client: HydroGoClient,
  type: string,
  queryClient: QueryClientLike,
  options?: ContentSubscribeOptions
): () => void {
  return client.subscribeContent<T>(
    type,
    (event) => {
      void queryClient.invalidateQueries({ queryKey: ["hydrogo", "content", type] });
      if (event.record && typeof (event.record as Record<string, unknown>).id === "string") {
        const recordId = (event.record as Record<string, unknown>).id as string;
        void queryClient.invalidateQueries({
          queryKey: ["hydrogo", "content", type, "record", recordId],
        });
      }
    },
    options
  );
}

export function pageQueryOptions(client: HydroGoClient, locale: string, slug: string) {
  return {
    queryKey: ["hydrogo", "pages", locale, slug] as const,
    queryFn: (): Promise<Page> => client.getPage(locale, slug),
  };
}

export function newsListQueryOptions(client: HydroGoClient, locale: string, limit = 20, offset = 0) {
  return {
    queryKey: ["hydrogo", "news", "list", locale, limit, offset] as const,
    queryFn: (): Promise<NewsPage> => client.listNewsPage(locale, limit, offset),
  };
}

export function newsItemQueryOptions(client: HydroGoClient, locale: string, slug: string) {
  return {
    queryKey: ["hydrogo", "news", "item", locale, slug] as const,
    queryFn: (): Promise<NewsItem> => client.getNews(locale, slug),
  };
}
