import { useState, useRef } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  Image,
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

export function PhotoCapture({ step, onCapture, capturedUri }: Props) {
  const [permission, requestPermission] = useCameraPermissions();
  const [showCamera, setShowCamera] = useState(false);
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
        <Text style={styles.description}>Camera access is required to take photos.</Text>
        <TouchableOpacity style={styles.button} onPress={requestPermission}>
          <Text style={styles.buttonText}>Grant Camera Access</Text>
        </TouchableOpacity>
      </View>
    );
  }

  if (showCamera) {
    return (
      <View style={styles.cameraContainer}>
        <CameraView ref={cameraRef} style={styles.camera} facing="back">
          <View style={styles.cameraOverlay}>
            <Text style={styles.cameraTitle}>{step.title}</Text>
          </View>
          <View style={styles.cameraControls}>
            <TouchableOpacity
              style={styles.cancelButton}
              onPress={() => setShowCamera(false)}
            >
              <Text style={styles.cancelText}>Cancel</Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={styles.captureButton}
              disabled={processing}
              onPress={async () => {
                if (!cameraRef.current || processing) return;
                setProcessing(true);
                try {
                  const photo = await cameraRef.current.takePictureAsync({
                    quality: 0.8,
                    exif: true,
                  });
                  if (!photo?.uri) {
                    Alert.alert('Error', 'Failed to capture photo.');
                    return;
                  }

                  // Save to app document directory
                  const filename = `photo_${Date.now()}.jpg`;
                  const destUri = `${documentDirectory}${filename}`;
                  await moveAsync({ from: photo.uri, to: destUri });

                  // Hash and get size
                  const [sha256, fileSize] = await Promise.all([
                    hashFile(destUri),
                    getFileSize(destUri),
                  ]);

                  onCapture({ uri: destUri, sha256, fileSize });
                  setShowCamera(false);
                } catch (err) {
                  Alert.alert('Error', 'Failed to process photo.');
                } finally {
                  setProcessing(false);
                }
              }}
            >
              {processing ? (
                <ActivityIndicator color="#fff" />
              ) : (
                <View style={styles.captureInner} />
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
          <Image source={{ uri: capturedUri }} style={styles.preview} />
          <View style={styles.previewActions}>
            <TouchableOpacity
              style={[styles.button, styles.retakeButton]}
              onPress={() => setShowCamera(true)}
            >
              <Text style={styles.retakeText}>Retake</Text>
            </TouchableOpacity>
          </View>
        </View>
      ) : (
        <TouchableOpacity style={styles.button} onPress={() => setShowCamera(true)}>
          <Text style={styles.buttonText}>Take Photo</Text>
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
  preview: {
    width: '100%',
    height: 250,
    borderRadius: 12,
    marginBottom: 12,
    backgroundColor: '#f3f4f6',
  },
  previewActions: {
    flexDirection: 'row',
    gap: 10,
  },
  retakeButton: {
    flex: 1,
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
  captureInner: {
    width: 58,
    height: 58,
    borderRadius: 29,
    backgroundColor: '#fff',
  },
});
