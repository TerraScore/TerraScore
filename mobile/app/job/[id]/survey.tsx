import { useEffect, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  Alert,
  ActivityIndicator,
  ScrollView,
} from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';
import NetInfo from '@react-native-community/netinfo';
import { api } from '@/services/api';
import { uploadMediaNative } from '@/services/upload';
import { enqueueUpload, saveSurveyDraft } from '@/services/db';
import { useLocation } from '@/hooks/useLocation';
import { useSurveyStore } from '@/stores/surveyStore';
import { StepRenderer } from '@/components/survey/StepRenderer';
import type { SurveyTemplate, ApiResponse } from '@/types/api';
import type { GPSPoint } from '@/stores/surveyStore';

export default function SurveyScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { location } = useLocation(10000);

  const store = useSurveyStore();
  const {
    steps,
    currentStepIndex,
    responses,
    gpsTrail,
    mediaQueue,
    startedAt,
    templateId,
    init,
    setResponse,
    addMedia,
    addGPSPoint,
    nextStep,
    prevStep,
    isAllComplete,
    reset,
  } = store;

  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load template on mount
  useEffect(() => {
    (async () => {
      try {
        const { data } = await api.get<ApiResponse<SurveyTemplate>>(
          `/v1/jobs/${id}/template`,
        );
        const template = data.data;
        if (!template) {
          setError('No survey template found for this job.');
          return;
        }
        init(id!, template.id, template.steps);
      } catch {
        setError('Failed to load survey template.');
      } finally {
        setLoading(false);
      }
    })();

    return () => {
      // Don't reset on unmount — allow returning to survey
    };
  }, [id]);

  const currentStep = steps[currentStepIndex];
  const isLast = currentStepIndex === steps.length - 1;
  const currentResponse = currentStep ? responses[currentStep.id] : undefined;

  const handleChecklistChange = (value: 'yes' | 'no' | 'na') => {
    if (!currentStep) return;
    setResponse(currentStep.id, {
      step_id: currentStep.id,
      type: 'checklist',
      value,
      completed: true,
    });
  };

  const handleMediaCapture = (data: { uri: string; sha256: string; fileSize: number }) => {
    if (!currentStep) return;
    const contentType = currentStep.type === 'video' ? 'video/mp4' : 'image/jpeg';
    setResponse(currentStep.id, {
      step_id: currentStep.id,
      type: currentStep.type,
      value: data,
      completed: true,
    });
    addMedia({
      step_id: currentStep.id,
      uri: data.uri,
      sha256: data.sha256,
      content_type: contentType,
      file_size: data.fileSize,
      lat: location?.lat ?? 0,
      lng: location?.lng ?? 0,
      captured_at: new Date().toISOString(),
      uploaded: false,
    });
  };

  const handleGPSPoint = (point: GPSPoint) => {
    addGPSPoint(point);
  };

  const handleGPSComplete = (geoJson: string) => {
    if (!currentStep) return;
    setResponse(currentStep.id, {
      step_id: currentStep.id,
      type: 'gps_trace',
      value: geoJson,
      completed: true,
    });
  };

  const saveOffline = async (
    responsesPayload: Record<string, any>,
    mediaToQueue: typeof mediaQueue,
  ) => {
    for (const media of mediaToQueue) {
      await enqueueUpload({
        jobId: id!,
        filePath: media.uri,
        contentType: media.content_type,
        stepId: media.step_id,
        sha256: media.sha256,
        fileSize: media.file_size,
        lat: media.lat,
        lng: media.lng,
        capturedAt: media.captured_at,
      });
    }
    await saveSurveyDraft({
      jobId: id!,
      templateId: templateId!,
      responsesJson: JSON.stringify(responsesPayload),
      gpsTrailJson: JSON.stringify(gpsTrail),
      startedAt,
    });
    reset();
  };

  const handleSubmit = async () => {
    if (!isAllComplete()) {
      Alert.alert('Incomplete', 'Please complete all required steps before submitting.');
      return;
    }

    setSubmitting(true);

    // Build payloads used by both online and offline paths
    const responsesPayload: Record<string, any> = {};
    for (const [stepId, resp] of Object.entries(responses)) {
      if (resp.type === 'checklist') {
        responsesPayload[stepId] = resp.value;
      } else if (resp.type === 'gps_trace') {
        responsesPayload[stepId] = 'completed';
      } else {
        responsesPayload[stepId] = 'uploaded';
      }
    }

    // Track which media have been uploaded so far (for partial failure recovery)
    const uploadedStepIds = new Set<string>();

    try {
      // Check network status
      const netState = await NetInfo.fetch();
      const isOnline = netState.isConnected === true && netState.isInternetReachable !== false;

      if (!isOnline) {
        const pendingMedia = mediaQueue.filter((m) => !m.uploaded);
        await saveOffline(responsesPayload, pendingMedia);
        Alert.alert('Saved Offline', 'Survey saved and will auto-submit when you reconnect.', [
          { text: 'OK', onPress: () => router.replace('/(tabs)/active') },
        ]);
        return;
      }

      // --- Online path: upload and submit immediately ---
      const pendingMedia = mediaQueue.filter((m) => !m.uploaded);
      for (const media of pendingMedia) {
        await uploadMediaNative({
          jobId: id!,
          stepId: media.step_id,
          uri: media.uri,
          contentType: media.content_type,
          sha256: media.sha256,
          fileSize: media.file_size,
          lat: media.lat,
          lng: media.lng,
          capturedAt: media.captured_at,
        });
        uploadedStepIds.add(media.step_id);
      }

      const trailGeoJson = JSON.stringify({
        type: 'LineString',
        coordinates: gpsTrail.map((p) => [p.lng, p.lat]),
      });

      const durationMinutes = startedAt
        ? (Date.now() - new Date(startedAt).getTime()) / 60000
        : undefined;

      await api.post(`/v1/jobs/${id}/survey`, {
        responses: responsesPayload,
        gps_trail_geojson: trailGeoJson,
        started_at: startedAt,
        duration_minutes: durationMinutes,
        template_id: templateId,
      });

      reset();
      Alert.alert('Success', 'Survey submitted successfully.', [
        { text: 'OK', onPress: () => router.replace('/(tabs)/active') },
      ]);
    } catch (err: any) {
      // Only queue media items that haven't been uploaded yet
      const failedMedia = mediaQueue.filter(
        (m) => !m.uploaded && !uploadedStepIds.has(m.step_id),
      );
      const msg = err?.response?.data?.error?.message ?? 'Failed to submit survey.';
      Alert.alert('Submit Failed', msg, [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Save Offline',
          onPress: async () => {
            await saveOffline(responsesPayload, failedMedia);
            router.replace('/(tabs)/active');
          },
        },
      ]);
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color="#2563eb" />
        <Text style={styles.loadingText}>Loading survey template...</Text>
      </View>
    );
  }

  if (error) {
    return (
      <View style={styles.center}>
        <Text style={styles.errorText}>{error}</Text>
        <TouchableOpacity style={styles.backLink} onPress={() => router.back()}>
          <Text style={styles.backLinkText}>Go back</Text>
        </TouchableOpacity>
      </View>
    );
  }

  if (!currentStep) {
    return (
      <View style={styles.center}>
        <Text style={styles.errorText}>No steps in template</Text>
      </View>
    );
  }

  const completedCount = Object.values(responses).filter((r) => r.completed).length;

  return (
    <View style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()}>
          <Text style={styles.backButton}>← Back</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle}>Survey</Text>
        <Text style={styles.progress}>
          {completedCount}/{steps.length} complete
        </Text>
      </View>

      {/* Progress bar */}
      <View style={styles.progressBar}>
        <View
          style={[
            styles.progressFill,
            { width: `${((currentStepIndex + 1) / steps.length) * 100}%` },
          ]}
        />
      </View>

      {/* Step indicator */}
      <View style={styles.stepIndicator}>
        <Text style={styles.stepLabel}>
          Step {currentStepIndex + 1} of {steps.length}
        </Text>
        {currentStep.required && <Text style={styles.requiredBadge}>Required</Text>}
      </View>

      {/* Step content */}
      <ScrollView style={styles.content} contentContainerStyle={styles.contentInner}>
        <StepRenderer
          step={currentStep}
          response={currentResponse}
          gpsPoints={gpsTrail}
          location={location}
          onChecklistChange={handleChecklistChange}
          onMediaCapture={handleMediaCapture}
          onGPSPoint={handleGPSPoint}
          onGPSComplete={handleGPSComplete}
        />
      </ScrollView>

      {/* Navigation */}
      <View style={styles.nav}>
        <TouchableOpacity
          style={[styles.navButton, currentStepIndex === 0 && styles.navButtonDisabled]}
          onPress={prevStep}
          disabled={currentStepIndex === 0}
        >
          <Text style={[styles.navButtonText, currentStepIndex === 0 && styles.navTextDisabled]}>
            Previous
          </Text>
        </TouchableOpacity>

        {isLast ? (
          <TouchableOpacity
            style={[styles.navButton, styles.submitButton, submitting && styles.navButtonDisabled]}
            onPress={handleSubmit}
            disabled={submitting}
          >
            {submitting ? (
              <ActivityIndicator color="#fff" size="small" />
            ) : (
              <Text style={styles.submitText}>Submit Survey</Text>
            )}
          </TouchableOpacity>
        ) : (
          <TouchableOpacity style={[styles.navButton, styles.nextButton]} onPress={nextStep}>
            <Text style={styles.nextText}>Next</Text>
          </TouchableOpacity>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f5f5f5',
  },
  center: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 32,
  },
  loadingText: {
    fontSize: 14,
    color: '#666',
    marginTop: 12,
  },
  errorText: {
    fontSize: 16,
    color: '#ef4444',
    textAlign: 'center',
    marginBottom: 16,
  },
  backLink: {
    paddingVertical: 8,
  },
  backLinkText: {
    color: '#2563eb',
    fontSize: 15,
  },
  header: {
    paddingHorizontal: 16,
    paddingTop: 60,
    paddingBottom: 12,
    backgroundColor: '#fff',
    flexDirection: 'row',
    alignItems: 'flex-end',
    justifyContent: 'space-between',
  },
  backButton: {
    fontSize: 16,
    color: '#2563eb',
    fontWeight: '500',
  },
  headerTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: '#1a1a2e',
  },
  progress: {
    fontSize: 13,
    color: '#666',
  },
  progressBar: {
    height: 3,
    backgroundColor: '#e5e7eb',
  },
  progressFill: {
    height: 3,
    backgroundColor: '#2563eb',
  },
  stepIndicator: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 10,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  stepLabel: {
    fontSize: 13,
    color: '#888',
  },
  requiredBadge: {
    fontSize: 11,
    color: '#ef4444',
    fontWeight: '600',
    backgroundColor: '#fef2f2',
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 4,
  },
  content: {
    flex: 1,
  },
  contentInner: {
    paddingBottom: 20,
  },
  nav: {
    flexDirection: 'row',
    gap: 12,
    padding: 16,
    backgroundColor: '#fff',
    borderTopWidth: 1,
    borderTopColor: '#eee',
  },
  navButton: {
    flex: 1,
    paddingVertical: 14,
    borderRadius: 10,
    alignItems: 'center',
    backgroundColor: '#f3f4f6',
  },
  navButtonDisabled: {
    opacity: 0.4,
  },
  navButtonText: {
    fontSize: 15,
    fontWeight: '600',
    color: '#666',
  },
  navTextDisabled: {
    color: '#bbb',
  },
  nextButton: {
    backgroundColor: '#2563eb',
  },
  nextText: {
    color: '#fff',
    fontSize: 15,
    fontWeight: '600',
  },
  submitButton: {
    backgroundColor: '#22c55e',
  },
  submitText: {
    color: '#fff',
    fontSize: 15,
    fontWeight: '600',
  },
});
