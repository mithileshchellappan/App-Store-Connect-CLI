import { FormEvent, startTransition, useEffect, useEffectEvent, useRef, useState } from "react";

import "./styles.css";
import {
  NavSection, ChatMessage, EnvSnapshot, StudioSettings, AuthState, AppStatusData,
  AppDetail, AppListItem, LocalizationEntry, ScreenshotSet,
  AppStatusState, TestFlightState, GroupTestersState, ReviewsState,
  SubscriptionsState, PricingOverviewState, FinanceRegionsState, OfferCodesState, FeedbackState,
} from "./types";
import {
  allSections, sectionCommands, appScopedSectionIDs,
  emptyEnv, defaultSettings, emptyAuthStatus,
} from "./constants";
import {
  normalizeEnvSnapshot, normalizeStudioSettings, normalizeAuthStatus,
  mapAppList, shellQuote, commandForApp, sectionRequiresApp, insightsWeekStart,
} from "./utils";
import { useTheme } from "./hooks/useTheme";
import { Sidebar } from "./components/Sidebar";
import { ContextBar } from "./components/ContextBar";
import { ChatDock } from "./components/ChatDock";
import { SettingsView } from "./components/views/SettingsView";
import { AppInfoView } from "./components/views/AppInfoView";
import { StatusView } from "./components/views/StatusView";
import { TestFlightView } from "./components/views/TestFlightView";
import { FeedbackView } from "./components/views/FeedbackView";
import { SubscriptionsView } from "./components/views/SubscriptionsView";
import { PricingView } from "./components/views/PricingView";
import { ReviewsView } from "./components/views/ReviewsView";
import { ScreenshotsView } from "./components/views/ScreenshotsView";
import { InsightsView } from "./components/views/InsightsView";
import { FinanceView } from "./components/views/FinanceView";
import { PromoCodesView } from "./components/views/PromoCodesView";
import { GenericTableView } from "./components/views/GenericTableView";
import { ToolView } from "./components/views/ToolView";
import { Bootstrap, CheckAuthStatus, GetAppDetail, GetFeedback, GetFinanceRegions, GetOfferCodes, GetPricingOverview, GetScreenshots, GetSettings, GetSubscriptions, GetTestFlight, GetTestFlightTesters, GetVersionMetadata, ListApps, RunASCCommand, SaveSettings } from "../wailsjs/go/main/App";
import { environment, settings as settingsNS } from "../wailsjs/go/models";

export { insightsWeekStart } from "./utils";

export default function App() {
  const [activeScope, setActiveScope] = useState<string>("app");
  const [activeSection, setActiveSection] = useState<NavSection>(allSections[0]);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [draft, setDraft] = useState("");
  const [dockExpanded, setDockExpanded] = useState(false);

  const [env, setEnv] = useState<EnvSnapshot>(emptyEnv as EnvSnapshot);
  const [studioSettings, setStudioSettings] = useState<StudioSettings>(defaultSettings as StudioSettings);
  const [settingsSaved, setSettingsSaved] = useState(false);
  const [bootstrapError, setBootstrapError] = useState("");
  const [loading, setLoading] = useState(true);
  const [authStatus, setAuthStatus] = useState<AuthState>(emptyAuthStatus as AuthState);
  const [appList, setAppList] = useState<AppListItem[]>([]);
  const [selectedAppId, setSelectedAppId] = useState<string | null>(null);
  const [appDetail, setAppDetail] = useState<AppDetail | null>(null);
  const [, setDetailLoading] = useState(false);
  const [allLocalizations, setAllLocalizations] = useState<LocalizationEntry[]>([]);
  const [selectedLocale, setSelectedLocale] = useState<string>("");
  const [metadataLoading, setMetadataLoading] = useState(false);
  const [screenshotSets, setScreenshotSets] = useState<ScreenshotSet[]>([]);
  const [screenshotsLoading, setScreenshotsLoading] = useState(false);
  const [appsLoading, setAppsLoading] = useState(false);
  const [sectionCache, setSectionCache] = useState<Record<string, { loading: boolean; error?: string; items: Record<string, unknown>[] }>>({});
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [appStatus, setAppStatus] = useState<AppStatusState>({ loading: false, data: null });
  const [testflightData, setTestflightData] = useState<TestFlightState>({ loading: false, groups: [] });
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null);
  const [groupTesters, setGroupTesters] = useState<GroupTestersState>({ loading: false, testers: [] });
  const [reviews, setReviews] = useState<ReviewsState>({ loading: false, items: [] });
  const [subscriptions, setSubscriptions] = useState<SubscriptionsState>({ loading: false, items: [] });
  const [pricingOverview, setPricingOverview] = useState<PricingOverviewState>({ loading: false, availableInNewTerritories: false, currentPrice: "", currentProceeds: "", baseCurrency: "", territories: [], subscriptionPricing: [] });
  const [selectedSub, setSelectedSub] = useState<string | null>(null);
  const [bundleIDsPlatformSort, setBundleIDsPlatformSort] = useState<"asc" | "desc">("asc");
  const [appSearchTerm, setAppSearchTerm] = useState("");
  const [sectionSearchTerms, setSectionSearchTerms] = useState<Record<string, string>>({});
  const [showBundleIDSheet, setShowBundleIDSheet] = useState(false);
  const [bundleIDName, setBundleIDName] = useState("");
  const [bundleIDIdentifier, setBundleIDIdentifier] = useState("");
  const [bundleIDPlatform, setBundleIDPlatform] = useState("IOS");
  const [bundleIDCreateError, setBundleIDCreateError] = useState("");
  const [bundleIDCreating, setBundleIDCreating] = useState(false);
  const [showDeviceSheet, setShowDeviceSheet] = useState(false);
  const [deviceName, setDeviceName] = useState("");
  const [deviceUDID, setDeviceUDID] = useState("");
  const [devicePlatform, setDevicePlatform] = useState("IOS");
  const [deviceCreateError, setDeviceCreateError] = useState("");
  const [deviceCreating, setDeviceCreating] = useState(false);
  const [financeRegions, setFinanceRegions] = useState<FinanceRegionsState>({ loading: false, regions: [] });
  const [offerCodes, setOfferCodes] = useState<OfferCodesState>({ loading: false, codes: [] });
  const [feedbackData, setFeedbackData] = useState<FeedbackState>({ loading: false, total: 0, items: [] });
  const appSelectionRequestRef = useRef(0);
  const screenshotRequestRef = useRef(0);
  const groupTesterRequestRef = useRef(0);
  const insightsRequestRef = useRef(0);
  const offerCodesRequestRef = useRef(0);

  const { resolvedTheme } = useTheme(studioSettings.theme);

  const loadStudioShell = useEffectEvent(async (options?: {
    clearApps?: boolean;
    isCancelled?: () => boolean;
  }) => {
    const isCancelled = options?.isCancelled ?? (() => false);

    try {
      const [data, auth] = await Promise.all([Bootstrap(), CheckAuthStatus()]);
      if (isCancelled()) return;

      startTransition(() => {
        setEnv(normalizeEnvSnapshot(data.environment));
        setStudioSettings(normalizeStudioSettings(data.settings));
        setAuthStatus(normalizeAuthStatus(auth));
        setBootstrapError("");
        if (options?.clearApps) {
          setAppList([]);
        }
      });

      if (!auth?.authenticated) {
        if (isCancelled()) return;
        startTransition(() => {
          setAppList([]);
          setAppsLoading(false);
        });
        return;
      }

      setAppsLoading(true);
      try {
        const res = await ListApps();
        if (isCancelled()) return;
        startTransition(() => {
          setAppList(mapAppList(res.apps));
        });
      } catch {
        if (isCancelled()) return;
        startTransition(() => {
          setAppList([]);
        });
      } finally {
        if (!isCancelled()) {
          setAppsLoading(false);
        }
      }
    } catch (err) {
      if (isCancelled()) return;
      setBootstrapError(String(err));
    } finally {
      if (!isCancelled()) {
        setLoading(false);
      }
    }
  });

  useEffect(() => {
    let cancelled = false;

    void loadStudioShell({ isCancelled: () => cancelled });
    return () => {
      cancelled = true;
    };
  }, []);

  function updateSetting<K extends keyof StudioSettings>(key: K, value: StudioSettings[K]) {
    setStudioSettings((prev) => ({ ...prev, [key]: value }));
    setSettingsSaved(false);
  }

  function handleSaveSettings() {
    const payload = new settingsNS.StudioSettings({
      preferredPreset: studioSettings.preferredPreset,
      agentCommand: studioSettings.agentCommand,
      agentArgs: studioSettings.agentArgs,
      agentEnv: studioSettings.agentEnv,
      preferBundledASC: studioSettings.preferBundledASC,
      systemASCPath: studioSettings.systemASCPath,
      workspaceRoot: studioSettings.workspaceRoot,
      theme: studioSettings.theme,
      windowMaterial: studioSettings.windowMaterial,
      showCommandPreviews: studioSettings.showCommandPreviews,
    });
    SaveSettings(payload)
      .then(() => setSettingsSaved(true))
      .catch((err) => console.error("save settings:", err));
  }

  // Prefetch all section data in parallel for an app
  function prefetchSections(appId: string, requestID: number) {
    const isStale = () => appSelectionRequestRef.current !== requestID;
    const quotedAppID = shellQuote(appId);
    setSectionCache((prev) => {
      const next = { ...prev };
      delete next.insights;
      for (const sectionId of appScopedSectionIDs) {
        next[sectionId] = { loading: true, items: [] };
      }
      return next;
    });
    // App status dashboard
    setAppStatus({ loading: true, data: null });
    RunASCCommand(`status --app ${quotedAppID} --output json`)
      .then((res) => {
        if (isStale()) return;
        if (res.error) { setAppStatus({ loading: false, error: res.error, data: null }); return; }
        try { setAppStatus({ loading: false, data: JSON.parse(res.data) }); }
        catch { setAppStatus({ loading: false, error: "Failed to parse status", data: null }); }
      })
      .catch((e) => {
        if (isStale()) return;
        setAppStatus({ loading: false, error: String(e), data: null });
      });

    // TestFlight groups with tester counts
    setTestflightData({ loading: true, groups: [] });
    setSelectedGroup(null);
    setGroupTesters({ loading: false, testers: [] });
    groupTesterRequestRef.current += 1;
    GetTestFlight(appId)
      .then((res) => {
        if (isStale()) return;
        if (res.error) setTestflightData({ loading: false, error: res.error, groups: [] });
        else setTestflightData({ loading: false, groups: res.groups ?? [] });
      })
      .catch((e) => {
        if (isStale()) return;
        setTestflightData({ loading: false, error: String(e), groups: [] });
      });

    // Reviews
    setReviews({ loading: true, items: [] });
    RunASCCommand(`reviews list --app ${quotedAppID} --limit 25 --output json`)
      .then((res) => {
        if (isStale()) return;
        if (res.error) { setReviews({ loading: false, error: res.error, items: [] }); return; }
        try {
          const d = JSON.parse(res.data);
          setReviews({ loading: false, items: (d.data ?? []).map((i: { attributes: Record<string, unknown> }) => i.attributes) });
        } catch { setReviews({ loading: false, error: "Failed to parse", items: [] }); }
      })
      .catch((e) => {
        if (isStale()) return;
        setReviews({ loading: false, error: String(e), items: [] });
      });

    // Pricing overview
    setPricingOverview({ loading: true, availableInNewTerritories: false, currentPrice: "", currentProceeds: "", baseCurrency: "", territories: [], subscriptionPricing: [] });
    GetPricingOverview(appId)
      .then((res) => {
        if (isStale()) return;
        if (res.error) setPricingOverview({ loading: false, error: res.error, availableInNewTerritories: false, currentPrice: "", currentProceeds: "", baseCurrency: "", territories: [], subscriptionPricing: [] });
        else setPricingOverview({ loading: false, availableInNewTerritories: res.availableInNewTerritories, currentPrice: res.currentPrice, currentProceeds: res.currentProceeds, baseCurrency: res.baseCurrency, territories: res.territories ?? [], subscriptionPricing: res.subscriptionPricing ?? [] });
      })
      .catch((e) => {
        if (isStale()) return;
        setPricingOverview({ loading: false, error: String(e), availableInNewTerritories: false, currentPrice: "", currentProceeds: "", baseCurrency: "", territories: [], subscriptionPricing: [] });
      });

    // Subscriptions: dedicated two-phase fetch
    setSubscriptions({ loading: true, items: [] });
    GetSubscriptions(appId)
      .then((res) => {
        if (isStale()) return;
        if (res.error) setSubscriptions({ loading: false, error: res.error, items: [] });
        else setSubscriptions({ loading: false, items: res.subscriptions ?? [] });
      })
      .catch((e) => {
        if (isStale()) return;
        setSubscriptions({ loading: false, error: String(e), items: [] });
      });

    // Finance regions
    setFinanceRegions({ loading: true, regions: [] });
    GetFinanceRegions()
      .then((res) => {
        if (isStale()) return;
        if (res.error) setFinanceRegions({ loading: false, error: res.error, regions: [] });
        else setFinanceRegions({ loading: false, regions: res.regions ?? [] });
      })
      .catch((e) => { if (!isStale()) setFinanceRegions({ loading: false, error: String(e), regions: [] }); });

    // TestFlight feedback
    setFeedbackData({ loading: true, total: 0, items: [] });
    GetFeedback(appId)
      .then((res) => {
        if (isStale()) return;
        if (res.error) setFeedbackData({ loading: false, error: res.error, total: 0, items: [] });
        else setFeedbackData({ loading: false, total: res.total, items: res.feedback ?? [] });
      })
      .catch((e) => { if (!isStale()) setFeedbackData({ loading: false, error: String(e), total: 0, items: [] }); });

    // Offer codes are loaded lazily when the promo-codes section is opened.
    setOfferCodes({ loading: false, loadedAppId: "", codes: [] });

    for (const [sectionId, cmdTemplate] of Object.entries(sectionCommands)) {
      if (!sectionRequiresApp(sectionId)) continue;
      const cmd = commandForApp(cmdTemplate, appId);
      RunASCCommand(cmd)
        .then((res) => {
          if (isStale()) return;
          if (res.error) {
            setSectionCache((prev) => ({ ...prev, [sectionId]: { loading: false, error: res.error, items: [] } }));
            return;
          }
          try {
            const parsed = JSON.parse(res.data);
            const items: Record<string, unknown>[] = [];
            if (Array.isArray(parsed?.data)) {
              for (const item of parsed.data) {
                items.push({ id: item.id, type: item.type, ...item.attributes });
              }
            } else if (parsed?.data?.attributes) {
              items.push({ id: parsed.data.id, type: parsed.data.type, ...parsed.data.attributes });
            }
            setSectionCache((prev) => ({ ...prev, [sectionId]: { loading: false, items } }));
          } catch {
            setSectionCache((prev) => ({ ...prev, [sectionId]: { loading: false, error: "Failed to parse response", items: [] } }));
          }
        })
        .catch((e) => {
          if (isStale()) return;
          setSectionCache((prev) => ({ ...prev, [sectionId]: { loading: false, error: String(e), items: [] } }));
        });
    }
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
          setSectionCache((prev) => ({ ...prev, [sectionId]: { loading: false, error: res.error, items: [] } }));
          return;
        }
        try {
          const parsed = JSON.parse(res.data);
          const items: Record<string, unknown>[] = [];
          if (Array.isArray(parsed?.data)) {
            for (const item of parsed.data) {
              items.push({ id: item.id, type: item.type, ...item.attributes });
            }
          } else if (parsed?.data?.attributes) {
            items.push({ id: parsed.data.id, type: parsed.data.type, ...parsed.data.attributes });
          }
          setSectionCache((prev) => ({ ...prev, [sectionId]: { loading: false, items } }));
        } catch {
          setSectionCache((prev) => ({ ...prev, [sectionId]: { loading: false, error: "Failed to parse response", items: [] } }));
        }
      })
      .catch((e) => {
        setSectionCache((prev) => ({ ...prev, [sectionId]: { loading: false, error: String(e), items: [] } }));
      });
  }

  const bundleIDCreateCommand =
    `bundle-ids create --identifier ${shellQuote(bundleIDIdentifier.trim())} --name ${shellQuote(bundleIDName.trim())} --platform ${bundleIDPlatform} --output json`;
  const deviceRegisterCommand =
    `devices register --name ${shellQuote(deviceName.trim())} --udid ${shellQuote(deviceUDID.trim())} --platform ${devicePlatform} --output json`;

  function closeBundleIDSheet() {
    setShowBundleIDSheet(false);
    setBundleIDCreateError("");
    setBundleIDCreating(false);
  }

  function resetBundleIDForm() {
    setBundleIDName("");
    setBundleIDIdentifier("");
    setBundleIDPlatform("IOS");
    setBundleIDCreateError("");
    setBundleIDCreating(false);
  }

  function openBundleIDSheet() {
    resetBundleIDForm();
    setShowBundleIDSheet(true);
  }

  function handleCreateBundleID() {
    const trimmedName = bundleIDName.trim();
    const trimmedIdentifier = bundleIDIdentifier.trim();
    if (!trimmedName || !trimmedIdentifier) {
      setBundleIDCreateError("Name and identifier are required.");
      return;
    }

    setBundleIDCreating(true);
    setBundleIDCreateError("");

    RunASCCommand(
      `bundle-ids create --identifier ${shellQuote(trimmedIdentifier)} --name ${shellQuote(trimmedName)} --platform ${bundleIDPlatform} --output json`,
    )
      .then((res) => {
        if (res.error) {
          setBundleIDCreateError(res.error);
          return;
        }
        closeBundleIDSheet();
        resetBundleIDForm();
        loadStandaloneSection("bundle-ids", true);
      })
      .catch((err) => {
        setBundleIDCreateError(String(err));
      })
      .finally(() => {
        setBundleIDCreating(false);
      });
  }

  function closeDeviceSheet() {
    setShowDeviceSheet(false);
    setDeviceCreateError("");
    setDeviceCreating(false);
  }

  function resetDeviceForm() {
    setDeviceName("");
    setDeviceUDID("");
    setDevicePlatform("IOS");
    setDeviceCreateError("");
    setDeviceCreating(false);
  }

  function openDeviceSheet() {
    resetDeviceForm();
    setShowDeviceSheet(true);
  }

  function handleCreateDevice() {
    const trimmedName = deviceName.trim();
    const trimmedUDID = deviceUDID.trim();
    if (!trimmedName || !trimmedUDID) {
      setDeviceCreateError("Name and UDID are required.");
      return;
    }

    setDeviceCreating(true);
    setDeviceCreateError("");

    RunASCCommand(
      `devices register --name ${shellQuote(trimmedName)} --udid ${shellQuote(trimmedUDID)} --platform ${devicePlatform} --output json`,
    )
      .then((res) => {
        if (res.error) {
          setDeviceCreateError(res.error);
          return;
        }
        closeDeviceSheet();
        resetDeviceForm();
        loadStandaloneSection("devices", true);
      })
      .catch((err) => {
        setDeviceCreateError(String(err));
      })
      .finally(() => {
        setDeviceCreating(false);
      });
  }

  function handleSelectApp(id: string) {
    const requestID = appSelectionRequestRef.current + 1;
    appSelectionRequestRef.current = requestID;
    screenshotRequestRef.current += 1;
    groupTesterRequestRef.current += 1;
    startTransition(() => {
      setSelectedAppId(id);
      setAppDetail(null);
      setAllLocalizations([]);
      setSelectedLocale("");
      setScreenshotSets([]);
      setSectionCache({});
      setSelectedGroup(null);
      setGroupTesters({ loading: false, testers: [] });
      setSelectedSub(null);
      setDetailLoading(true);
    });
    // Fire all section prefetches in parallel
    prefetchSections(id, requestID);
    GetAppDetail(id)
      .then((d) => {
        if (appSelectionRequestRef.current !== requestID) return;
        const detail = {
          id: d.id, name: d.name, subtitle: d.subtitle, bundleId: d.bundleId,
          sku: d.sku, primaryLocale: d.primaryLocale, versions: d.versions ?? [], error: d.error,
        };
        setAppDetail(detail);
        // Fetch metadata for the primary iOS version (fallback to first version)
        const primaryVersion = (d.versions ?? []).find((v: { platform: string }) => v.platform === "IOS")
          ?? (d.versions ?? [])[0];
        if (primaryVersion?.id) {
          setMetadataLoading(true);
          GetVersionMetadata(primaryVersion.id)
            .then((meta) => {
              if (appSelectionRequestRef.current !== requestID) return;
              if (meta.localizations?.length) {
                setAllLocalizations(meta.localizations);
                const defaultLoc = meta.localizations.find(
                  (l: { locale: string }) => l.locale === d.primaryLocale
                ) ?? meta.localizations[0];
                setSelectedLocale(defaultLoc.locale);
                // Fetch screenshots for the default locale in parallel
                if (defaultLoc.localizationId) {
                  const screenshotRequestID = screenshotRequestRef.current + 1;
                  screenshotRequestRef.current = screenshotRequestID;
                  setScreenshotsLoading(true);
                  GetScreenshots(defaultLoc.localizationId)
                    .then((res) => {
                      if (appSelectionRequestRef.current !== requestID || screenshotRequestRef.current !== screenshotRequestID) return;
                      setScreenshotSets(res.sets ?? []);
                    })
                    .catch(() => {})
                    .finally(() => {
                      if (appSelectionRequestRef.current !== requestID || screenshotRequestRef.current !== screenshotRequestID) return;
                      setScreenshotsLoading(false);
                    });
                }
              }
            })
            .catch(() => {})
            .finally(() => {
              if (appSelectionRequestRef.current !== requestID) return;
              setMetadataLoading(false);
            });
        }
      })
      .catch((e) => {
        if (appSelectionRequestRef.current !== requestID) return;
        setAppDetail({ id, name: "", subtitle: "", bundleId: "", sku: "", primaryLocale: "", versions: [], error: String(e) });
      })
      .finally(() => {
        if (appSelectionRequestRef.current !== requestID) return;
        setDetailLoading(false);
      });
  }

  function handleLocaleChange(locale: string) {
    setSelectedLocale(locale);
    const loc = allLocalizations.find((l) => l.locale === locale);
    if (loc?.localizationId) {
      const screenshotRequestID = screenshotRequestRef.current + 1;
      screenshotRequestRef.current = screenshotRequestID;
      setScreenshotsLoading(true);
      setScreenshotSets([]);
      GetScreenshots(loc.localizationId)
        .then((res) => {
          if (screenshotRequestRef.current !== screenshotRequestID) return;
          setScreenshotSets(res.sets ?? []);
        })
        .catch(() => {})
        .finally(() => {
          if (screenshotRequestRef.current !== screenshotRequestID) return;
          setScreenshotsLoading(false);
        });
    }
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = draft.trim();
    if (!trimmed) return;

    setMessages((current) => [
      ...current,
      { id: `user-${current.length}`, role: "user", content: trimmed, timestamp: "Now" },
      {
        id: `assistant-${current.length}`,
        role: "assistant",
        content: "Bootstrap mode recorded the prompt. Live ACP transport is not wired yet.",
        timestamp: "Now",
      },
    ]);
    setDraft("");
    setDockExpanded(true);
  }

  const handleRefresh = useEffectEvent(() => {
    if (selectedAppId) {
      handleSelectApp(selectedAppId);
    } else {
      setLoading(true);
      setBootstrapError("");
      void loadStudioShell({ clearApps: true });
    }
  });

  // Cmd+R to refresh
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === "r") {
        e.preventDefault();
        handleRefresh();
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  useEffect(() => {
    if (!sectionCommands[activeSection.id]) return;
    if (sectionRequiresApp(activeSection.id)) return;
    if (sectionCache[activeSection.id]) return;
    loadStandaloneSection(activeSection.id);
  }, [activeSection, sectionCache]);

  useEffect(() => {
    if (activeSection.id !== "promo-codes" || !selectedAppId) return;
    if (offerCodes.loading || offerCodes.loadedAppId === selectedAppId) return;

    const appRequestID = appSelectionRequestRef.current;
    const offerRequestID = offerCodesRequestRef.current + 1;
    offerCodesRequestRef.current = offerRequestID;

    setOfferCodes({ loading: true, loadedAppId: selectedAppId, codes: [] });
    GetOfferCodes(selectedAppId)
      .then((res) => {
        if (
          appSelectionRequestRef.current !== appRequestID ||
          offerCodesRequestRef.current !== offerRequestID
        ) {
          return;
        }
        if (res.error) {
          setOfferCodes({ loading: false, loadedAppId: selectedAppId, error: res.error, codes: [] });
          return;
        }
        setOfferCodes({ loading: false, loadedAppId: selectedAppId, codes: res.offerCodes ?? [] });
      })
      .catch((error) => {
        if (
          appSelectionRequestRef.current !== appRequestID ||
          offerCodesRequestRef.current !== offerRequestID
        ) {
          return;
        }
        setOfferCodes({ loading: false, loadedAppId: selectedAppId, error: String(error), codes: [] });
      });
  }, [activeSection.id, offerCodes.loadedAppId, offerCodes.loading, selectedAppId]);

  useEffect(() => {
    if (activeSection.id !== "insights" || !selectedAppId) return;
    if (sectionCache.insights) return;

    const weekStr = insightsWeekStart(new Date());
    const appRequestID = appSelectionRequestRef.current;
    const insightsRequestID = insightsRequestRef.current + 1;
    insightsRequestRef.current = insightsRequestID;

    setSectionCache((prev) => ({
      ...prev,
      insights: { loading: true, items: [] },
    }));

    void RunASCCommand(
      `insights weekly --app ${shellQuote(selectedAppId)} --source analytics --week ${weekStr} --output json`,
    )
      .then((res) => {
        if (
          appSelectionRequestRef.current !== appRequestID ||
          insightsRequestRef.current !== insightsRequestID
        ) {
          return;
        }
        if (res.error) {
          setSectionCache((prev) => ({ ...prev, insights: { loading: false, error: res.error, items: [] } }));
          return;
        }
        try {
          const d = JSON.parse(res.data);
          const metrics = (d.metrics ?? []).map((m: Record<string, unknown>) => m);
          setSectionCache((prev) => ({ ...prev, insights: { loading: false, items: metrics } }));
        } catch {
          setSectionCache((prev) => ({ ...prev, insights: { loading: false, error: "Failed to parse", items: [] } }));
        }
      })
      .catch((error) => {
        if (
          appSelectionRequestRef.current !== appRequestID ||
          insightsRequestRef.current !== insightsRequestID
        ) {
          return;
        }
        setSectionCache((prev) => ({ ...prev, insights: { loading: false, error: String(error), items: [] } }));
      });
  }, [activeSection.id, sectionCache.insights, selectedAppId]);

  const authConfigured = authStatus.authenticated;
  const insightsWeek = insightsWeekStart(new Date());
  const insightsCache = sectionCache.insights;
  const filteredApps = appList.filter((app) =>
    `${app.name} ${app.subtitle}`.toLowerCase().includes(appSearchTerm.trim().toLowerCase()),
  );
  const activeSectionSearch = sectionSearchTerms[activeSection.id] ?? "";

  function handleSelectGroup(groupId: string) {
    const testerRequestID = groupTesterRequestRef.current + 1;
    groupTesterRequestRef.current = testerRequestID;
    setSelectedGroup(groupId);
    setGroupTesters({ loading: true, testers: [] });
    GetTestFlightTesters(groupId)
      .then((res) => {
        if (groupTesterRequestRef.current !== testerRequestID) return;
        setGroupTesters({ loading: false, testers: res.testers ?? [] });
      })
      .catch(() => {
        if (groupTesterRequestRef.current !== testerRequestID) return;
        setGroupTesters({ loading: false, testers: [] });
      });
  }

  // Tool view configurations
  const toolViews: Record<string, { title: string; description: string; commandHint: string }> = {
    diff: {
      title: "Diff",
      description: "Generate deterministic diff plans between app versions.",
      commandHint: `Use the ACP chat to run: asc diff metadata --app ${selectedAppId || "APP_ID"}`,
    },
    actors: {
      title: "Actors",
      description: "Actors are users and API keys that appear in audit fields (e.g. submittedByActor). Look up an actor by ID:",
      commandHint: "asc actors view --id ACTOR_ID",
    },
    migrate: {
      title: "Migrate",
      description: "Migrate from Fastlane to asc.",
      commandHint: "Use ACP chat: asc migrate import --fastfile ./Fastfile",
    },
    notify: {
      title: "Notifications",
      description: "Send notifications via Slack or webhooks.",
      commandHint: "Use ACP chat: asc notify slack --webhook $WEBHOOK --message \"Build ready\"",
    },
    notarization: {
      title: "Notarization",
      description: "Submit macOS apps for Apple notarization.",
      commandHint: "Use the ACP chat to run: asc notarization submit --file ./MyApp.zip",
    },
    crashes: {
      title: "Crashes",
      description: "Crash diagnostics are per-build. Select a build ID from the Builds section to view diagnostics.",
      commandHint: `asc performance diagnostics list --build BUILD_ID --output json`,
    },
    "app-setup": {
      title: "App Setup",
      description: "Post-create app configuration: locale, categories, availability, pricing.",
      commandHint: `asc app-setup info set --app ${selectedAppId || "APP_ID"} --primary-locale "en-US"`,
    },
    "routing-coverage": {
      title: "Routing Coverage",
      description: "Routing app coverage files require a version ID.",
      commandHint: `asc routing-coverage view --version-id VERSION_ID`,
    },
    "build-localizations": {
      title: "Build Localizations",
      description: "Release notes per build. Requires a build ID.",
      commandHint: `asc build-localizations list --build BUILD_ID --output json`,
    },
    "build-bundles": {
      title: "Build Bundles",
      description: "Build bundle information. Requires a build ID.",
      commandHint: `asc build-bundles list --build BUILD_ID --output json`,
    },
    schema: {
      title: "Schema",
      description: "Browse the App Store Connect API schema.",
      commandHint: "asc schema index --output json",
    },
    metadata: {
      title: "Metadata",
      description: "Pull and push app metadata.",
      commandHint: `asc metadata pull --app ${selectedAppId || "APP_ID"} --dir ./metadata`,
    },
    agreements: {
      title: "Agreements",
      description: "Territory agreements for EULA. Requires an EULA ID.",
      commandHint: "asc agreements territories list --id EULA_ID --output json",
    },
    workflow: {
      title: "Workflow",
      description: "List and run asc workflows.",
      commandHint: "asc workflow list",
    },
  };

  // Render the main content area
  function renderContent() {
    if (loading) {
      return (
        <div className="empty-state">
          <p className="empty-hint">Loading…</p>
        </div>
      );
    }
    if (bootstrapError) {
      return (
        <div className="empty-state">
          <p className="empty-title">Bootstrap failed</p>
          <p className="empty-hint">{bootstrapError}</p>
        </div>
      );
    }
    if (activeSection.id === "settings") {
      return (
        <SettingsView
          authStatus={authStatus}
          env={env}
          studioSettings={studioSettings}
          settingsSaved={settingsSaved}
          updateSetting={updateSetting}
          handleSaveSettings={handleSaveSettings}
        />
      );
    }
    if (!authConfigured) {
      return (
        <div className="empty-state">
          <p className="empty-title">No credentials configured</p>
          <p className="empty-hint">
            Run <code>asc init</code> to create an API key profile, or go to Settings to check your configuration.
          </p>
          <button
            className="toolbar-btn"
            type="button"
            onClick={() => setActiveSection(allSections.find((s) => s.id === "settings")!)}
          >
            Open Settings
          </button>
        </div>
      );
    }
    if (activeSection.id === "overview" && appDetail) {
      return (
        <AppInfoView
          appDetail={appDetail}
          selectedAppId={selectedAppId}
          metadataLoading={metadataLoading}
          allLocalizations={allLocalizations}
          selectedLocale={selectedLocale}
          screenshotsLoading={screenshotsLoading}
          screenshotSets={screenshotSets}
          onLocaleChange={handleLocaleChange}
          onRunCommand={RunASCCommand}
        />
      );
    }
    if (activeSection.id === "status" && selectedAppId) {
      return <StatusView appStatus={appStatus} />;
    }
    if (activeSection.id === "testflight" && selectedAppId) {
      return (
        <TestFlightView
          testflightData={testflightData}
          selectedGroup={selectedGroup}
          groupTesters={groupTesters}
          onSelectGroup={handleSelectGroup}
          onBackToGroups={() => setSelectedGroup(null)}
        />
      );
    }
    if (activeSection.id === "insights" && selectedAppId) {
      return <InsightsView insightsWeek={insightsWeek} insightsCache={insightsCache} />;
    }
    if (activeSection.id === "finance" && selectedAppId) {
      return <FinanceView financeRegions={financeRegions} />;
    }
    if (activeSection.id === "pricing" && selectedAppId) {
      return <PricingView pricingOverview={pricingOverview} />;
    }
    if (activeSection.id === "subscriptions" && selectedAppId) {
      return (
        <SubscriptionsView
          subscriptions={subscriptions}
          selectedSub={selectedSub}
          onSelectSub={setSelectedSub}
        />
      );
    }
    if (activeSection.id === "ratings-reviews" && selectedAppId) {
      return <ReviewsView reviews={reviews} />;
    }
    if (activeSection.id === "screenshots" && selectedAppId) {
      return (
        <ScreenshotsView
          screenshotsLoading={screenshotsLoading}
          screenshotSets={screenshotSets}
          allLocalizations={allLocalizations}
          selectedLocale={selectedLocale}
          onLocaleChange={handleLocaleChange}
        />
      );
    }
    if (activeSection.id === "feedback" && selectedAppId) {
      return <FeedbackView feedbackData={feedbackData} />;
    }
    if (activeSection.id === "promo-codes" && selectedAppId) {
      return <PromoCodesView offerCodes={offerCodes} />;
    }
    // Tool views (diff, actors, migrate, notify, notarization)
    if (toolViews[activeSection.id]) {
      const tv = toolViews[activeSection.id];
      return <ToolView title={tv.title} description={tv.description} commandHint={tv.commandHint} />;
    }
    // Generic table/card views for sections backed by sectionCommands
    if (sectionCommands[activeSection.id] && (!sectionRequiresApp(activeSection.id) || selectedAppId)) {
      const cache = sectionCache[activeSection.id];
      return (
        <GenericTableView
          activeSection={activeSection}
          cache={cache ?? { loading: true, items: [] }}
          bundleIDsPlatformSort={bundleIDsPlatformSort}
          activeSectionSearch={activeSectionSearch}
          onSetSectionSearch={(sectionId, term) => setSectionSearchTerms((prev) => ({ ...prev, [sectionId]: term }))}
          onToggleBundleIDSort={() => setBundleIDsPlatformSort((prev) => prev === "asc" ? "desc" : "asc")}
          onOpenBundleIDSheet={openBundleIDSheet}
          onOpenDeviceSheet={openDeviceSheet}
        />
      );
    }
    // Default empty state
    return (
      <div className="empty-state">
        <p className="empty-title">
          {!selectedAppId && activeSection.id !== "settings" ? "Select an App" : activeSection.label}
        </p>
        <p className="empty-hint">
          {!selectedAppId && activeSection.id !== "settings"
            ? "Use search in the sidebar to pick an app."
            : ""}
        </p>
      </div>
    );
  }

  return (
    <div className="studio-shell" data-theme={resolvedTheme}>
      <a href="#main-content" className="skip-nav">Skip to main content</a>
      <Sidebar
        activeScope={activeScope}
        selectedAppId={selectedAppId}
        appDetail={appDetail}
        appList={appList}
        appSearchTerm={appSearchTerm}
        activeSection={activeSection}
        appsLoading={appsLoading}
        authAuthenticated={authStatus.authenticated}
        filteredApps={filteredApps}
        onAppSearchChange={setAppSearchTerm}
        onSelectApp={handleSelectApp}
        onSetActiveSection={setActiveSection}
      />

      <div className="shell-separator" />

      {/* Main area */}
      <main id="main-content" className="main-area">
        <ContextBar
          authStatus={authStatus}
          activeScope={activeScope}
          handleRefresh={handleRefresh}
          setActiveScope={setActiveScope}
          setActiveSection={setActiveSection}
        />

        {renderContent()}

        {/* Chat dock — hidden on settings */}
        {activeSection.id !== "settings" && (
          <ChatDock
            messages={messages}
            draft={draft}
            dockExpanded={dockExpanded}
            handleSubmit={handleSubmit}
            setDraft={setDraft}
            setDockExpanded={setDockExpanded}
          />
        )}
      </main>

      {showBundleIDSheet && (
        <div className="sheet-backdrop" role="presentation" onClick={closeBundleIDSheet}>
          <section
            className="sheet-panel"
            role="dialog"
            aria-modal="true"
            aria-labelledby="bundle-id-sheet-title"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="sheet-header">
              <div>
                <p className="sheet-eyebrow">Signing</p>
                <h2 id="bundle-id-sheet-title" className="sheet-title">Create Bundle ID</h2>
              </div>
              <button type="button" className="sheet-close" onClick={closeBundleIDSheet} aria-label="Close create bundle ID sheet">
                &times;
              </button>
            </div>

            <div className="sheet-body">
              <label className="sheet-field">
                <span className="sheet-label">Name</span>
                <input
                  type="text"
                  value={bundleIDName}
                  onChange={(event) => setBundleIDName(event.target.value)}
                  placeholder="Example App"
                />
              </label>

              <label className="sheet-field">
                <span className="sheet-label">Identifier</span>
                <input
                  type="text"
                  value={bundleIDIdentifier}
                  onChange={(event) => setBundleIDIdentifier(event.target.value)}
                  placeholder="com.example.app"
                />
              </label>

              <label className="sheet-field">
                <span className="sheet-label">Platform</span>
                <select value={bundleIDPlatform} onChange={(event) => setBundleIDPlatform(event.target.value)}>
                  <option value="IOS">iOS</option>
                  <option value="MAC_OS">macOS</option>
                  <option value="TV_OS">tvOS</option>
                  <option value="VISION_OS">visionOS</option>
                </select>
              </label>

              <div className="sheet-preview">
                <p className="sheet-label">Command preview</p>
                <code>{bundleIDCreateCommand}</code>
              </div>

              {bundleIDCreateError && <p className="sheet-error">{bundleIDCreateError}</p>}
            </div>

            <div className="sheet-footer">
              <button type="button" className="toolbar-btn" onClick={closeBundleIDSheet}>
                Cancel
              </button>
              <button
                type="button"
                className="toolbar-btn toolbar-btn-primary"
                onClick={handleCreateBundleID}
                disabled={bundleIDCreating}
              >
                {bundleIDCreating ? "Creating…" : "Create"}
              </button>
            </div>
          </section>
        </div>
      )}

      {showDeviceSheet && (
        <div className="sheet-backdrop" role="presentation" onClick={closeDeviceSheet}>
          <section
            className="sheet-panel"
            role="dialog"
            aria-modal="true"
            aria-labelledby="device-sheet-title"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="sheet-header">
              <div>
                <p className="sheet-eyebrow">Team</p>
                <h2 id="device-sheet-title" className="sheet-title">Register Device</h2>
              </div>
              <button type="button" className="sheet-close" onClick={closeDeviceSheet} aria-label="Close register device sheet">
                &times;
              </button>
            </div>

            <div className="sheet-body">
              <label className="sheet-field">
                <span className="sheet-label">Name</span>
                <input
                  type="text"
                  value={deviceName}
                  onChange={(event) => setDeviceName(event.target.value)}
                  placeholder="Rudrank's iPhone"
                />
              </label>

              <label className="sheet-field">
                <span className="sheet-label">UDID</span>
                <input
                  type="text"
                  value={deviceUDID}
                  onChange={(event) => setDeviceUDID(event.target.value)}
                  placeholder="00008110-001234560E90003A"
                />
              </label>

              <label className="sheet-field">
                <span className="sheet-label">Platform</span>
                <select value={devicePlatform} onChange={(event) => setDevicePlatform(event.target.value)}>
                  <option value="IOS">iOS</option>
                  <option value="MAC_OS">macOS</option>
                  <option value="TV_OS">tvOS</option>
                  <option value="VISION_OS">visionOS</option>
                </select>
              </label>

              <div className="sheet-preview">
                <p className="sheet-label">Command preview</p>
                <code>{deviceRegisterCommand}</code>
              </div>

              {deviceCreateError && <p className="sheet-error">{deviceCreateError}</p>}
            </div>

            <div className="sheet-footer">
              <button type="button" className="toolbar-btn" onClick={closeDeviceSheet}>
                Cancel
              </button>
              <button
                type="button"
                className="toolbar-btn toolbar-btn-primary"
                onClick={handleCreateDevice}
                disabled={deviceCreating}
              >
                {deviceCreating ? "Registering…" : "Register"}
              </button>
            </div>
          </section>
        </div>
      )}
    </div>
  );
}
