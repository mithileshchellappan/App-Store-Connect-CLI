import { useRef, useState } from "react";

import { useAppMetadata } from "./appSelection/useAppMetadata";
import { useAppSectionData } from "./appSelection/useAppSectionData";
import { useTestFlightData } from "./appSelection/useTestFlightData";

export function useAppSelection() {
  const [selectedAppId, setSelectedAppId] = useState<string | null>(null);
  const [selectionVersion, setSelectionVersion] = useState(0);

  const appSelectionRequestRef = useRef(0);
  const metadata = useAppMetadata(appSelectionRequestRef);
  const sectionData = useAppSectionData(appSelectionRequestRef);
  const testFlight = useTestFlightData(appSelectionRequestRef);

  function handleSelectApp(id: string) {
    const requestID = appSelectionRequestRef.current + 1;
    appSelectionRequestRef.current = requestID;

    setSelectionVersion(requestID);
    setSelectedAppId(id);
    metadata.resetSelection();
    sectionData.resetSelection();
    testFlight.resetSelection();

    metadata.loadAppDetail(id, requestID);
  }

  return {
    selectionVersion,
    selectedAppId,
    appDetail: metadata.appDetail,
    allLocalizations: metadata.allLocalizations,
    selectedLocale: metadata.selectedLocale,
    metadataLoading: metadata.metadataLoading,
    metadataError: metadata.metadataError,
    screenshotSets: metadata.screenshotSets,
    screenshotsLoading: metadata.screenshotsLoading,
    screenshotsError: metadata.screenshotsError,
    sectionCache: sectionData.sectionCache,
    appStatus: sectionData.appStatus,
    reviews: sectionData.reviews,
    subscriptions: sectionData.subscriptions,
    pricingOverview: sectionData.pricingOverview,
    selectedSub: sectionData.selectedSub,
    financeRegions: sectionData.financeRegions,
    offerCodes: sectionData.offerCodes,
    feedbackData: sectionData.feedbackData,
    testflightData: testFlight.testflightData,
    selectedGroup: testFlight.selectedGroup,
    groupTesters: testFlight.groupTesters,
    setSelectedSub: sectionData.setSelectedSub,
    handleSelectApp,
    handleLocaleChange: metadata.handleLocaleChange,
    handleSelectGroup: testFlight.handleSelectGroup,
    handleBackToGroups: testFlight.handleBackToGroups,
    loadAppSectionIfNeeded: sectionData.loadAppSectionIfNeeded,
    loadStandaloneSection: sectionData.loadStandaloneSection,
    loadStandaloneSectionIfNeeded: sectionData.loadStandaloneSectionIfNeeded,
    loadTestFlightIfNeeded: (appId: string, force = false) =>
      testFlight.loadGroupsIfNeeded(appId, appSelectionRequestRef.current, force),
    loadOfferCodesIfNeeded: sectionData.loadOfferCodesIfNeeded,
    loadInsightsIfNeeded: sectionData.loadInsightsIfNeeded,
  };
}
