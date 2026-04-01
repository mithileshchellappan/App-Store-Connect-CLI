import { useRef, useState, type MutableRefObject } from "react";

import {
  AppStatusState,
  FeedbackState,
  FinanceRegionsState,
  OfferCodesState,
  PricingOverviewState,
  ReviewsState,
  SectionCacheEntry,
  SubscriptionsState,
} from "../../types";
import { appScopedSectionIDs, sectionCommands } from "../../constants";
import { commandForApp, insightsWeekStart, sectionRequiresApp, shellQuote } from "../../utils";
import {
  GetFeedback,
  GetFinanceRegions,
  GetOfferCodes,
  GetPricingOverview,
  GetSubscriptions,
  RunASCCommand,
} from "../../../wailsjs/go/main/App";
import { parseCommandItems } from "./commandItems";

function emptyAppStatus(): AppStatusState {
  return { loading: false, data: null };
}

function emptyReviews(): ReviewsState {
  return { loading: false, items: [] };
}

function emptySubscriptions(): SubscriptionsState {
  return { loading: false, items: [] };
}

function emptyPricingOverview(): PricingOverviewState {
  return {
    loading: false,
    availableInNewTerritories: false,
    currentPrice: "",
    currentProceeds: "",
    baseCurrency: "",
    territories: [],
    subscriptionPricing: [],
  };
}

function emptyFinanceRegions(): FinanceRegionsState {
  return { loading: false, regions: [] };
}

function emptyOfferCodes(): OfferCodesState {
  return { loading: false, loadedAppId: "", codes: [] };
}

function emptyFeedback(): FeedbackState {
  return { loading: false, total: 0, items: [] };
}

export function useAppSectionData(appSelectionRequestRef: MutableRefObject<number>) {
  const [sectionCache, setSectionCache] = useState<Record<string, SectionCacheEntry>>({});
  const [appStatus, setAppStatus] = useState<AppStatusState>(emptyAppStatus());
  const [reviews, setReviews] = useState<ReviewsState>(emptyReviews());
  const [subscriptions, setSubscriptions] = useState<SubscriptionsState>(emptySubscriptions());
  const [pricingOverview, setPricingOverview] = useState<PricingOverviewState>(emptyPricingOverview());
  const [selectedSub, setSelectedSub] = useState<string | null>(null);
  const [financeRegions, setFinanceRegions] = useState<FinanceRegionsState>(emptyFinanceRegions());
  const [offerCodes, setOfferCodes] = useState<OfferCodesState>(emptyOfferCodes());
  const [feedbackData, setFeedbackData] = useState<FeedbackState>(emptyFeedback());

  const insightsRequestRef = useRef(0);
  const offerCodesRequestRef = useRef(0);

  function resetSelection() {
    insightsRequestRef.current += 1;
    offerCodesRequestRef.current += 1;
    setSectionCache((prev) => {
      const next = { ...prev };
      delete next.insights;
      for (const sectionId of appScopedSectionIDs) {
        delete next[sectionId];
      }
      return next;
    });
    setAppStatus(emptyAppStatus());
    setReviews(emptyReviews());
    setSubscriptions(emptySubscriptions());
    setPricingOverview(emptyPricingOverview());
    setSelectedSub(null);
    setFinanceRegions(emptyFinanceRegions());
    setOfferCodes(emptyOfferCodes());
    setFeedbackData(emptyFeedback());
  }

  function loadStatusIfNeeded(appId: string, force = false) {
    if (!force && (appStatus.loading || (appStatus.loadedAppId === appId && !appStatus.error))) return;

    const requestID = appSelectionRequestRef.current;
    setAppStatus({ loading: true, loadedAppId: appId, data: null });

    RunASCCommand(`status --app ${shellQuote(appId)} --output json`)
      .then((res) => {
        if (appSelectionRequestRef.current !== requestID) return;
        if (res.error) {
          setAppStatus({ loading: false, loadedAppId: appId, error: res.error, data: null });
          return;
        }
        try {
          setAppStatus({ loading: false, loadedAppId: appId, data: JSON.parse(res.data) });
        } catch {
          setAppStatus({ loading: false, loadedAppId: appId, error: "Failed to parse status", data: null });
        }
      })
      .catch((error) => {
        if (appSelectionRequestRef.current !== requestID) return;
        setAppStatus({ loading: false, loadedAppId: appId, error: String(error), data: null });
      });
  }

  function loadReviewsIfNeeded(appId: string, force = false) {
    if (!force && (reviews.loading || (reviews.loadedAppId === appId && !reviews.error))) return;

    const requestID = appSelectionRequestRef.current;
    setReviews({ loading: true, loadedAppId: appId, items: [] });

    RunASCCommand(`reviews list --app ${shellQuote(appId)} --limit 25 --output json`)
      .then((res) => {
        if (appSelectionRequestRef.current !== requestID) return;
        if (res.error) {
          setReviews({ loading: false, loadedAppId: appId, error: res.error, items: [] });
          return;
        }
        try {
          const parsed = JSON.parse(res.data);
          setReviews({
            loading: false,
            loadedAppId: appId,
            items: (parsed.data ?? []).map(
              (item: { attributes: Record<string, unknown> }) => item.attributes,
            ),
          });
        } catch {
          setReviews({ loading: false, loadedAppId: appId, error: "Failed to parse", items: [] });
        }
      })
      .catch((error) => {
        if (appSelectionRequestRef.current !== requestID) return;
        setReviews({ loading: false, loadedAppId: appId, error: String(error), items: [] });
      });
  }

  function loadPricingOverviewIfNeeded(appId: string, force = false) {
    if (
      !force &&
      (pricingOverview.loading || (pricingOverview.loadedAppId === appId && !pricingOverview.error))
    ) {
      return;
    }

    const requestID = appSelectionRequestRef.current;
    setPricingOverview({ ...emptyPricingOverview(), loading: true, loadedAppId: appId });

    GetPricingOverview(appId)
      .then((res) => {
        if (appSelectionRequestRef.current !== requestID) return;
        if (res.error) {
          setPricingOverview({
            ...emptyPricingOverview(),
            loading: false,
            loadedAppId: appId,
            error: res.error,
          });
          return;
        }
        setPricingOverview({
          loading: false,
          loadedAppId: appId,
          availableInNewTerritories: res.availableInNewTerritories,
          currentPrice: res.currentPrice,
          currentProceeds: res.currentProceeds,
          baseCurrency: res.baseCurrency,
          territories: res.territories ?? [],
          subscriptionPricing: res.subscriptionPricing ?? [],
        });
      })
      .catch((error) => {
        if (appSelectionRequestRef.current !== requestID) return;
        setPricingOverview({
          ...emptyPricingOverview(),
          loading: false,
          loadedAppId: appId,
          error: String(error),
        });
      });
  }

  function loadSubscriptionsIfNeeded(appId: string, force = false) {
    if (!force && (subscriptions.loading || (subscriptions.loadedAppId === appId && !subscriptions.error))) {
      return;
    }

    const requestID = appSelectionRequestRef.current;
    setSubscriptions({ loading: true, loadedAppId: appId, items: [] });

    GetSubscriptions(appId)
      .then((res) => {
        if (appSelectionRequestRef.current !== requestID) return;
        if (res.error) {
          setSubscriptions({
            loading: false,
            loadedAppId: appId,
            error: res.error,
            items: res.subscriptions ?? [],
          });
          return;
        }
        setSubscriptions({ loading: false, loadedAppId: appId, items: res.subscriptions ?? [] });
      })
      .catch((error) => {
        if (appSelectionRequestRef.current !== requestID) return;
        setSubscriptions({ loading: false, loadedAppId: appId, error: String(error), items: [] });
      });
  }

  function loadFinanceRegionsIfNeeded(appId: string, force = false) {
    if (
      !force &&
      (financeRegions.loading || (financeRegions.loadedAppId === appId && !financeRegions.error))
    ) {
      return;
    }

    const requestID = appSelectionRequestRef.current;
    setFinanceRegions({ loading: true, loadedAppId: appId, regions: [] });

    GetFinanceRegions()
      .then((res) => {
        if (appSelectionRequestRef.current !== requestID) return;
        if (res.error) {
          setFinanceRegions({ loading: false, loadedAppId: appId, error: res.error, regions: [] });
          return;
        }
        setFinanceRegions({ loading: false, loadedAppId: appId, regions: res.regions ?? [] });
      })
      .catch((error) => {
        if (appSelectionRequestRef.current !== requestID) return;
        setFinanceRegions({ loading: false, loadedAppId: appId, error: String(error), regions: [] });
      });
  }

  function loadFeedbackIfNeeded(appId: string, force = false) {
    if (!force && (feedbackData.loading || (feedbackData.loadedAppId === appId && !feedbackData.error))) {
      return;
    }

    const requestID = appSelectionRequestRef.current;
    setFeedbackData({ ...emptyFeedback(), loading: true, loadedAppId: appId });

    GetFeedback(appId)
      .then((res) => {
        if (appSelectionRequestRef.current !== requestID) return;
        if (res.error) {
          setFeedbackData({
            ...emptyFeedback(),
            loading: false,
            loadedAppId: appId,
            error: res.error,
          });
          return;
        }
        setFeedbackData({
          loading: false,
          loadedAppId: appId,
          total: res.total,
          items: res.feedback ?? [],
        });
      })
      .catch((error) => {
        if (appSelectionRequestRef.current !== requestID) return;
        setFeedbackData({
          ...emptyFeedback(),
          loading: false,
          loadedAppId: appId,
          error: String(error),
        });
      });
  }

  function loadAppScopedSectionIfNeeded(sectionId: string, appId: string, force = false) {
    const cmdTemplate = sectionCommands[sectionId];
    if (!cmdTemplate || !sectionRequiresApp(sectionId)) return;

    const requestID = appSelectionRequestRef.current;
    setSectionCache((prev) => {
      const existing = prev[sectionId];
      if (existing && !force) return prev;
      return { ...prev, [sectionId]: { loading: true, items: [] } };
    });

    RunASCCommand(commandForApp(cmdTemplate, appId))
      .then((res) => {
        if (appSelectionRequestRef.current !== requestID) return;
        if (res.error) {
          setSectionCache((prev) => ({
            ...prev,
            [sectionId]: { loading: false, error: res.error, items: [] },
          }));
          return;
        }
        try {
          setSectionCache((prev) => ({
            ...prev,
            [sectionId]: { loading: false, items: parseCommandItems(res.data) },
          }));
        } catch {
          setSectionCache((prev) => ({
            ...prev,
            [sectionId]: { loading: false, error: "Failed to parse response", items: [] },
          }));
        }
      })
      .catch((error) => {
        if (appSelectionRequestRef.current !== requestID) return;
        setSectionCache((prev) => ({
          ...prev,
          [sectionId]: { loading: false, error: String(error), items: [] },
        }));
      });
  }

  function loadStandaloneSection(sectionId: string, force = false) {
    const cmd = sectionCommands[sectionId];
    if (!cmd || sectionRequiresApp(sectionId)) return;

    setSectionCache((prev) => {
      const existing = prev[sectionId];
      if (existing && !force) return prev;
      return { ...prev, [sectionId]: { loading: true, items: [] } };
    });

    RunASCCommand(cmd)
      .then((res) => {
        if (res.error) {
          setSectionCache((prev) => ({
            ...prev,
            [sectionId]: { loading: false, error: res.error, items: [] },
          }));
          return;
        }
        try {
          setSectionCache((prev) => ({
            ...prev,
            [sectionId]: { loading: false, items: parseCommandItems(res.data) },
          }));
        } catch {
          setSectionCache((prev) => ({
            ...prev,
            [sectionId]: { loading: false, error: "Failed to parse response", items: [] },
          }));
        }
      })
      .catch((error) => {
        setSectionCache((prev) => ({
          ...prev,
          [sectionId]: { loading: false, error: String(error), items: [] },
        }));
      });
  }

  function loadStandaloneSectionIfNeeded(sectionId: string) {
    const existing = sectionCache[sectionId];
    if (existing?.loading) return;
    if (existing && !existing.error) return;
    loadStandaloneSection(sectionId, Boolean(existing?.error));
  }

  function loadOfferCodesIfNeeded(sectionId: string, appId: string | null, force = false) {
    if (sectionId !== "promo-codes" || !appId) return;
    if (!force && (offerCodes.loading || (offerCodes.loadedAppId === appId && !offerCodes.error))) return;

    const appRequestID = appSelectionRequestRef.current;
    const offerRequestID = offerCodesRequestRef.current + 1;
    offerCodesRequestRef.current = offerRequestID;

    setOfferCodes({ loading: true, loadedAppId: appId, codes: [] });

    GetOfferCodes(appId)
      .then((res) => {
        if (
          appSelectionRequestRef.current !== appRequestID ||
          offerCodesRequestRef.current !== offerRequestID
        ) {
          return;
        }
        if (res.error) {
          setOfferCodes({
            loading: false,
            loadedAppId: appId,
            error: res.error,
            codes: res.offerCodes ?? [],
          });
          return;
        }
        setOfferCodes({ loading: false, loadedAppId: appId, codes: res.offerCodes ?? [] });
      })
      .catch((error) => {
        if (
          appSelectionRequestRef.current !== appRequestID ||
          offerCodesRequestRef.current !== offerRequestID
        ) {
          return;
        }
        setOfferCodes({ loading: false, loadedAppId: appId, error: String(error), codes: [] });
      });
  }

  function loadInsightsIfNeeded(sectionId: string, appId: string | null, force = false) {
    if (sectionId !== "insights" || !appId) return;
    const existingInsights = sectionCache.insights;
    if (!force && (existingInsights?.loading || (existingInsights && !existingInsights.error))) return;

    const weekStr = insightsWeekStart(new Date());
    const appRequestID = appSelectionRequestRef.current;
    const insightsRequestID = insightsRequestRef.current + 1;
    insightsRequestRef.current = insightsRequestID;

    setSectionCache((prev) => ({ ...prev, insights: { loading: true, items: [] } }));

    void RunASCCommand(
      `insights weekly --app ${shellQuote(appId)} --source analytics --week ${weekStr} --output json`,
    )
      .then((res) => {
        if (
          appSelectionRequestRef.current !== appRequestID ||
          insightsRequestRef.current !== insightsRequestID
        ) {
          return;
        }
        if (res.error) {
          setSectionCache((current) => ({
            ...current,
            insights: { loading: false, error: res.error, items: [] },
          }));
          return;
        }
        try {
          const parsed = JSON.parse(res.data);
          setSectionCache((current) => ({
            ...current,
            insights: {
              loading: false,
              items: (parsed.metrics ?? []).map((metric: Record<string, unknown>) => metric),
            },
          }));
        } catch {
          setSectionCache((current) => ({
            ...current,
            insights: { loading: false, error: "Failed to parse", items: [] },
          }));
        }
      })
      .catch((error) => {
        if (
          appSelectionRequestRef.current !== appRequestID ||
          insightsRequestRef.current !== insightsRequestID
        ) {
          return;
        }
        setSectionCache((current) => ({
          ...current,
          insights: { loading: false, error: String(error), items: [] },
        }));
      });
  }

  function loadAppSectionIfNeeded(sectionId: string, appId: string | null, force = false) {
    if (!appId) return;

    if (sectionId === "status") {
      loadStatusIfNeeded(appId, force);
      return;
    }
    if (sectionId === "ratings-reviews") {
      loadReviewsIfNeeded(appId, force);
      return;
    }
    if (sectionId === "pricing") {
      loadPricingOverviewIfNeeded(appId, force);
      return;
    }
    if (sectionId === "subscriptions") {
      loadSubscriptionsIfNeeded(appId, force);
      return;
    }
    if (sectionId === "finance") {
      loadFinanceRegionsIfNeeded(appId, force);
      return;
    }
    if (sectionId === "feedback") {
      loadFeedbackIfNeeded(appId, force);
      return;
    }
    if (sectionId === "promo-codes") {
      loadOfferCodesIfNeeded(sectionId, appId, force);
      return;
    }
    if (sectionId === "insights") {
      loadInsightsIfNeeded(sectionId, appId, force);
      return;
    }

    loadAppScopedSectionIfNeeded(sectionId, appId, force);
  }

  return {
    sectionCache,
    appStatus,
    reviews,
    subscriptions,
    pricingOverview,
    selectedSub,
    financeRegions,
    offerCodes,
    feedbackData,
    setSelectedSub,
    setSectionCache,
    resetSelection,
    loadAppSectionIfNeeded,
    loadStandaloneSection,
    loadStandaloneSectionIfNeeded,
    loadOfferCodesIfNeeded,
    loadInsightsIfNeeded,
  };
}
