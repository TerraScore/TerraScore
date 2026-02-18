import { useState, useRef } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { CameraView, useCameraPermissions } from 'expo-camera';
import { documentDirectory, moveAsync } from 'expo-file-system/legacy';
import type { SurveyStep } from '@/types/api';
import { hashFile, getFileSize } from '@/services/upload';

interface Props {
  step: SurveyStep;
  onCapture: (data: {
    uri: string;
    sha256: string;
    fileSize: number;
  }) => void;
  capturedUri: string | null;
}

export function VideoCapture({ step, onCapture, capturedUri }: Props) {
  const [permission, requestPermission] = useCameraPermissions();
  const [showCamera, setShowCamera] = useState(false);
  const [recording, setRecording] = useState(false);
  const [processing, setProcessing] = useState(false);
  const cameraRef = useRef<CameraView>(null);

  if (!permission) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color="#2563eb" />
      </View>
    );
  }

  if (!permission.granted) {
    return (
      <View style={styles.container}>
        <Text style={styles.title}>{step.title}</Text>
        <Text style={styles.description}>Camera access is required to record video.</Text>
        <TouchableOpacity style={styles.button} onPress={requestPermission}>
          <Text style={styles.buttonText}>Grant Camera Access</Text>
        </TouchableOpacity>
      </View>
    );
  }

  if (showCamera) {
    return (
      <View style={styles.cameraContainer}>
        <CameraView ref={cameraRef} style={styles.camera} facing="back" mode="video">
          <View style={styles.cameraOverlay}>
            <Text style={styles.cameraTitle}>{step.title}</Text>
            {recording && (
              <View style={styles.recordingBadge}>
                <View style={styles.recordingDot} />
                <Text style={styles.recordingText}>Recording</Text>
              </View>
            )}
          </View>
          <View style={styles.cameraControls}>
            <TouchableOpacity
              style={styles.cancelButton}
              onPress={() => {
                if (recording) return;
                setShowCamera(false);
              }}
            >
              <Text style={styles.cancelText}>{recording ? '' : 'Cancel'}</Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={[styles.captureButton, recording && styles.captureButtonRecording]}
              disabled={processing}
              onPress={async () => {
                if (!cameraRef.current) return;

                if (recording) {
                  // Stop recording
                  setRecording(false);
                  setProcessing(true);
                  try {
                    cameraRef.current.stopRecording();
                  } catch {
                    setProcessing(false);
                  }
                } else {
                  // Start recording
                  setRecording(true);
                  try {
                    const video = await cameraRef.current.recordAsync({
                      maxDuration: 60,
                    });
                    if (!video?.uri) {
                      Alert.alert('Error', 'Failed to record video.');
                      setProcessing(false);
                      return;
                    }

                    const filename = `video_${Date.now()}.mp4`;
                    const destUri = `${documentDirectory}${filename}`;
                    await moveAsync({ from: video.uri, to: destUri });

                    const [sha256, fileSize] = await Promise.all([
                      hashFile(destUri),
                      getFileSize(destUri),
                    ]);

                    onCapture({ uri: destUri, sha256, fileSize });
                    setShowCamera(false);
                  } catch {
                    Alert.alert('Error', 'Failed to process video.');
                  } finally {
                    setRecording(false);
                    setProcessing(false);
                  }
                }
              }}
            >
              {processing ? (
                <ActivityIndicator color="#fff" />
              ) : recording ? (
                <View style={styles.stopInner} />
              ) : (
                <View style={styles.recordInner} />
              )}
            </TouchableOpacity>
            <View style={{ width: 60 }} />
          </View>
        </CameraView>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Text style={styles.title}>{step.title}</Text>
      {step.description && <Text style={styles.description}>{step.description}</Text>}

      {capturedUri ? (
        <View>
          <View style={styles.videoPreview}>
            <Text style={styles.videoPreviewText}>Video captured</Text>
          </View>
          <TouchableOpacity
            style={[styles.button, styles.retakeButton]}
            onPress={() => setShowCamera(true)}
          >
            <Text style={styles.retakeText}>Re-record</Text>
          </TouchableOpacity>
        </View>
      ) : (
        <TouchableOpacity style={styles.button} onPress={() => setShowCamera(true)}>
          <Text style={styles.buttonText}>Record Video</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    padding: 16,
  },
  center: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  title: {
    fontSize: 17,
    fontWeight: '600',
    color: '#1a1a2e',
    marginBottom: 6,
  },
  description: {
    fontSize: 14,
    color: '#666',
    marginBottom: 16,
  },
  button: {
    backgroundColor: '#2563eb',
    paddingVertical: 14,
    borderRadius: 10,
    alignItems: 'center',
  },
  buttonText: {
    color: '#fff',
    fontSize: 15,
    fontWeight: '600',
  },
  videoPreview: {
    height: 120,
    borderRadius: 12,
    backgroundColor: '#1a1a2e',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 12,
  },
  videoPreviewText: {
    color: '#fff',
    fontSize: 14,
  },
  retakeButton: {
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: '#e5e7eb',
  },
  retakeText: {
    color: '#666',
    fontSize: 15,
    fontWeight: '600',
  },
  cameraContainer: {
    flex: 1,
  },
  camera: {
    flex: 1,
  },
  cameraOverlay: {
    paddingTop: 60,
    paddingHorizontal: 16,
  },
  cameraTitle: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
    textShadowColor: 'rgba(0,0,0,0.5)',
    textShadowOffset: { width: 0, height: 1 },
    textShadowRadius: 3,
  },
  recordingBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: 8,
  },
  recordingDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: '#ef4444',
    marginRight: 6,
  },
  recordingText: {
    color: '#ef4444',
    fontSize: 13,
    fontWeight: '600',
  },
  cameraControls: {
    position: 'absolute',
    bottom: 40,
    left: 0,
    right: 0,
    flexDirection: 'row',
    justifyContent: 'space-around',
    alignItems: 'center',
  },
  cancelButton: {
    width: 60,
    alignItems: 'center',
  },
  cancelText: {
    color: '#fff',
    fontSize: 15,
    fontWeight: '500',
  },
  captureButton: {
    width: 72,
    height: 72,
    borderRadius: 36,
    backgroundColor: 'rgba(255,255,255,0.3)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  captureButtonRecording: {
    backgroundColor: 'rgba(239,68,68,0.3)',
  },
  recordInner: {
    width: 58,
    height: 58,
    borderRadius: 29,
    backgroundColor: '#ef4444',
  },
  stopInner: {
    width: 28,
    height: 28,
    borderRadius: 4,
    backgroundColor: '#ef4444',
  },
});
