package cmd

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/strangelove-ventures/horcrux-proxy/signer"
	cometprotoprivval "github.com/strangelove-ventures/horcrux/v3/comet/proto/privval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// MockHorcruxConnection implements signer.HorcruxConnection for testing
type MockHorcruxConnection struct {
	mock.Mock
}

func (m *MockHorcruxConnection) SendRequest(request cometprotoprivval.Message) (*cometprotoprivval.Message, error) {
	args := m.Called(request)
	return args.Get(0).(*cometprotoprivval.Message), args.Error(1)
}

// MockLogger implements cometlog.Logger for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, keyvals ...interface{}) {
	m.Called(msg, keyvals)
}

func (m *MockLogger) Info(msg string, keyvals ...interface{}) {
	m.Called(msg, keyvals)
}

func (m *MockLogger) Warn(msg string, keyvals ...interface{}) {
	m.Called(msg, keyvals)
}

func (m *MockLogger) Error(msg string, keyvals ...interface{}) {
	m.Called(msg, keyvals)
}

// setupMockSentryWatcher creates a SentryWatcher with mocked dependencies for testing
func setupMockSentryWatcher(t *testing.T, all bool, operator bool, sentries []string) (*SentryWatcher, *MockHorcruxConnection, *MockLogger, *fake.Clientset) {
	mockHC := new(MockHorcruxConnection)
	mockLogger := new(MockLogger)

	// Configure the logger to accept any message with any arguments
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("With", mock.Anything).Return(mockLogger)

	// Configure the horcrux connection to handle any requests
	mockHC.On("SendRequest", mock.Anything).Return(&cometprotoprivval.Message{}, nil)

	// Create a fake Kubernetes clientset
	clientset := fake.NewSimpleClientset()

	// Create a SentryWatcher with our mocks
	sw := &SentryWatcher{
		all:                all,
		client:             clientset,
		done:               make(chan struct{}),
		hc:                 mockHC,
		labels:             labelCosmosSentry,
		log:                mockLogger,
		node:               "test-node",
		operator:           operator,
		persistentSentries: make([]*signer.ReconnRemoteSigner, 0),
		sentries:           make(map[string]*signer.ReconnRemoteSigner),
		stop:               make(chan struct{}),
	}

	// Add persistent sentries if provided
	for _, sentryAddr := range sentries {
		dialer := net.Dialer{Timeout: 2 * time.Second}
		sw.persistentSentries = append(sw.persistentSentries, signer.NewReconnRemoteSigner(sentryAddr, mockLogger, mockHC, dialer, 1024))
	}

	return sw, mockHC, mockLogger, clientset
}

// Helper to create test pods and services
func createTestPodsAndServices(t *testing.T, clientset *fake.Clientset) {
	// Create namespaces
	_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create pods
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentry-pod-1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app":                         "sentry-1",
				"app.kubernetes.io/component": "cosmos-sentry",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "test-node",
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentry-pod-2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app":                         "sentry-2",
				"app.kubernetes.io/component": "cosmos-sentry",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "other-node",
		},
	}

	_, err = clientset.CoreV1().Pods("test-namespace").Create(context.Background(), pod1, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.CoreV1().Pods("test-namespace").Create(context.Background(), pod2, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create services
	service1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentry-service-1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/component": "cosmos-sentry",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "sentry-1",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "sentry-privval",
					Port: 1234,
				},
			},
		},
	}

	service2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentry-service-2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/component": "cosmos-sentry",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "sentry-2",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "sentry-privval",
					Port: 1234,
				},
			},
		},
	}

	_, err = clientset.CoreV1().Services("test-namespace").Create(context.Background(), service1, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.CoreV1().Services("test-namespace").Create(context.Background(), service2, metav1.CreateOptions{})
	require.NoError(t, err)
}

// safeCleanup safely cleans up the SentryWatcher without hanging
func safeCleanup(sw *SentryWatcher) {
	// Close the stop channel directly to signal any goroutines to exit
	close(sw.stop)

	// Give a little time for goroutines to clean up
	time.Sleep(100 * time.Millisecond)
}

func TestNewSentryWatcher_WithoutOperator(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test creating SentryWatcher without operator mode
	mockLogger := new(MockLogger)
	mockHC := new(MockHorcruxConnection)

	// Configure the logger to accept any message with any arguments
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("With", mock.Anything).Return(mockLogger)

	// Configure the horcrux connection to handle any requests
	mockHC.On("SendRequest", mock.Anything).Return(&cometprotoprivval.Message{}, nil)

	sentries := []string{"tcp://sentry1:1234", "tcp://sentry2:1234"}

	sw, err := NewSentryWatcher(
		ctx,
		[]string{},
		mockLogger,
		false,
		mockHC,
		false, // Not in operator mode
		sentries,
		1024,
	)

	require.NoError(t, err)
	assert.NotNil(t, sw)
	assert.Equal(t, false, sw.operator)
	assert.Equal(t, false, sw.all)
	assert.Len(t, sw.persistentSentries, 2)

	// Safely clean up
	safeCleanup(sw)
}

func TestReconcileSentries(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create SentryWatcher with mocked dependencies
	sw, _, _, clientset := setupMockSentryWatcher(t, false, true, []string{})

	// Create test pods and services
	createTestPodsAndServices(t, clientset)

	// Run reconcileSentries
	err := sw.reconcileSentries(ctx, 1024)
	require.NoError(t, err)

	// Should only connect to the service on the same node
	assert.Len(t, sw.sentries, 1)

	// Check if the correct sentry is connected
	hasSentry1 := false
	hasSentry2 := false
	for addr := range sw.sentries {
		if addr == "tcp://sentry-service-1.test-namespace:1234" {
			hasSentry1 = true
		}
		if addr == "tcp://sentry-service-2.test-namespace:1234" {
			hasSentry2 = true
		}
	}
	assert.True(t, hasSentry1)
	assert.False(t, hasSentry2) // Should not have sentry 2 as it's on a different node

	// Run reconcileSentries again - this should not add any new sentries
	err = sw.reconcileSentries(ctx, 1024)
	require.NoError(t, err)
	assert.Len(t, sw.sentries, 1)

	// Now create a new service and verify it gets added
	service3 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentry-service-3",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/component": "cosmos-sentry",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "sentry-3",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "sentry-privval",
					Port: 1234,
				},
			},
		},
	}

	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentry-pod-3",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "sentry-3",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "test-node",
		},
	}

	_, err = clientset.CoreV1().Pods("test-namespace").Create(context.Background(), pod3, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.CoreV1().Services("test-namespace").Create(context.Background(), service3, metav1.CreateOptions{})
	require.NoError(t, err)

	err = sw.reconcileSentries(ctx, 1024)
	require.NoError(t, err)
	assert.Len(t, sw.sentries, 2)

	// Safely clean up
	safeCleanup(sw)
}

func TestReconcileSentries_WithLabelSelector(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create SentryWatcher with mocked dependencies
	sw, _, _, clientset := setupMockSentryWatcher(t, true, true, []string{})

	// Create test pods and services
	createTestPodsAndServices(t, clientset)

	// Add a service with additional labels
	service3 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentry-service-3",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/component": "cosmos-sentry",
				"environment":                 "production",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "sentry-3",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "sentry-privval",
					Port: 1234,
				},
			},
		},
	}

	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentry-pod-3",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "sentry-3",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "test-node",
		},
	}

	_, err := clientset.CoreV1().Pods("test-namespace").Create(context.Background(), pod3, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.CoreV1().Services("test-namespace").Create(context.Background(), service3, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create a label selector that only matches service3
	sw.labels = "environment=production"

	// Run reconcileSentries
	err = sw.reconcileSentries(ctx, 1024)
	require.NoError(t, err)

	// Should only connect to service3
	assert.Len(t, sw.sentries, 1)

	// Check if the correct sentry is connected
	hasSentry3 := false
	for addr := range sw.sentries {
		if addr == "tcp://sentry-service-3.test-namespace:1234" {
			hasSentry3 = true
		}
	}
	assert.True(t, hasSentry3)

	// Safely clean up
	safeCleanup(sw)
}

func TestWatch_WithoutOperator(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test Watch method with operator=false
	sw, _, _, _ := setupMockSentryWatcher(t, false, false, []string{"tcp://persistent:1234"})

	// With operator=false, Watch should just start the persistent sentries and return
	go sw.Watch(ctx, 1024)

	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)

	// Safely clean up
	safeCleanup(sw)
}

func TestWatch_WithOperator(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test Watch method with operator=true
	sw, _, _, clientset := setupMockSentryWatcher(t, true, true, []string{"tcp://persistent:1234"})

	// Create test pods and services
	createTestPodsAndServices(t, clientset)

	// Start watching in a goroutine
	go sw.Watch(ctx, 1024)

	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)

	// Cancel the context to stop the watch
	cancel()

	// Safely clean up
	safeCleanup(sw)
}

func TestReconcileSentries_RemoveSentries(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test that sentries are properly removed when they no longer exist in Kubernetes
	sw, _, _, clientset := setupMockSentryWatcher(t, true, true, []string{})

	// Create test pods and services
	createTestPodsAndServices(t, clientset)

	// Run reconcileSentries to add the sentries
	err := sw.reconcileSentries(ctx, 1024)
	require.NoError(t, err)

	// Should have connected to both services (since all=true)
	assert.Len(t, sw.sentries, 2)

	// Store which sentries we have initially
	hasSentry1 := false
	hasSentry2 := false
	for addr := range sw.sentries {
		if addr == "tcp://sentry-service-1.test-namespace:1234" {
			hasSentry1 = true
		}
		if addr == "tcp://sentry-service-2.test-namespace:1234" {
			hasSentry2 = true
		}
	}
	assert.True(t, hasSentry1)
	assert.True(t, hasSentry2)

	// Now delete a service
	err = clientset.CoreV1().Services("test-namespace").Delete(ctx, "sentry-service-1", metav1.DeleteOptions{})
	require.NoError(t, err)

	// Run reconcileSentries again
	err = sw.reconcileSentries(ctx, 1024)
	require.NoError(t, err)

	// Should have removed one sentry
	assert.Len(t, sw.sentries, 1)

	// Verify the correct sentry remains
	hasSentry1 = false
	hasSentry2 = false
	for addr := range sw.sentries {
		if addr == "tcp://sentry-service-1.test-namespace:1234" {
			hasSentry1 = true
		}
		if addr == "tcp://sentry-service-2.test-namespace:1234" {
			hasSentry2 = true
		}
	}
	assert.False(t, hasSentry1)
	assert.True(t, hasSentry2)

	// Safely clean up
	safeCleanup(sw)
}

func TestNewSentryWatcher_UniqueLabels(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test that duplicate labels are removed
	mockLogger := new(MockLogger)
	mockHC := new(MockHorcruxConnection)

	// Configure the logger and horcrux connection
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("With", mock.Anything).Return(mockLogger)
	mockHC.On("SendRequest", mock.Anything).Return(&cometprotoprivval.Message{}, nil)

	labels := []string{
		"app.kubernetes.io/component=cosmos-sentry", // Should be deduplicated
		"environment=production",
		"app.kubernetes.io/component=cosmos-sentry",
	}

	sw, err := NewSentryWatcher(
		ctx,
		labels,
		mockLogger,
		false,
		mockHC,
		false,
		[]string{},
		1024,
	)

	require.NoError(t, err)

	// The order of labels may vary, so let's check they're all there
	assert.Contains(t, sw.labels, "environment=production")
	assert.Contains(t, sw.labels, "app.kubernetes.io/component=cosmos-sentry")
	assert.NotContains(t, sw.labels, "app.kubernetes.io/component=cosmos-sentry,app.kubernetes.io/component=cosmos-sentry")

	// Safely clean up
	safeCleanup(sw)
}

func TestReconcileSentries_SkipInvalidServices(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test that services without the correct port name are skipped
	sw, _, _, clientset := setupMockSentryWatcher(t, true, true, []string{})

	// Create a test namespace
	_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create a service with the wrong port name
	invalidService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-service",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/component": "cosmos-sentry",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "invalid",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "wrong-port-name",
					Port: 1234,
				},
			},
		},
	}

	_, err = clientset.CoreV1().Services("test-namespace").Create(context.Background(), invalidService, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create a pod for this service
	invalidPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-pod",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "invalid",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "test-node",
		},
	}
	_, err = clientset.CoreV1().Pods("test-namespace").Create(context.Background(), invalidPod, metav1.CreateOptions{})
	require.NoError(t, err)

	// Run reconcileSentries
	err = sw.reconcileSentries(ctx, 1024)
	require.NoError(t, err)

	// Should not have connected to any services
	assert.Len(t, sw.sentries, 0)

	// Safely clean up
	safeCleanup(sw)
}

func TestReconcileSentries_SkipMultiplePods(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test that services selecting multiple pods are skipped
	sw, _, _, clientset := setupMockSentryWatcher(t, true, true, []string{})

	// Create a test namespace
	_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create pods with the same label
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "same-app",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "test-node",
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "same-app",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "test-node",
		},
	}

	_, err = clientset.CoreV1().Pods("test-namespace").Create(context.Background(), pod1, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.CoreV1().Pods("test-namespace").Create(context.Background(), pod2, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create a service that selects both pods
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multi-pod-service",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/component": "cosmos-sentry",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "same-app",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "sentry-privval",
					Port: 1234,
				},
			},
		},
	}

	_, err = clientset.CoreV1().Services("test-namespace").Create(context.Background(), service, metav1.CreateOptions{})
	require.NoError(t, err)

	// Run reconcileSentries
	err = sw.reconcileSentries(ctx, 1024)
	require.NoError(t, err)

	// Should not have connected to any services
	assert.Len(t, sw.sentries, 0)

	// Safely clean up
	safeCleanup(sw)
}

func TestReconcileSentries_WithParseToLabelSelector(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create SentryWatcher with mocked dependencies
	sw, _, _, clientset := setupMockSentryWatcher(t, true, true, []string{})

	// Create test pods and services
	createTestPodsAndServices(t, clientset)

	// Add a service with additional labels
	service3 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentry-service-3",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/component": "cosmos-sentry",
				"chain":                       "osmosis",
				"network":                     "mainnet",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "sentry-3",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "sentry-privval",
					Port: 1234,
				},
			},
		},
	}

	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentry-pod-3",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "sentry-3",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "test-node",
		},
	}

	_, err := clientset.CoreV1().Pods("test-namespace").Create(context.Background(), pod3, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.CoreV1().Services("test-namespace").Create(context.Background(), service3, metav1.CreateOptions{})
	require.NoError(t, err)

	sw.labels = "chain=osmosis,network=mainnet"

	// Run reconcileSentries with the selector
	err = sw.reconcileSentries(ctx, 1024)
	require.NoError(t, err)

	// Should only connect to service3
	assert.Len(t, sw.sentries, 1)

	// Check if the correct sentry is connected
	hasSentry3 := false
	for addr := range sw.sentries {
		if addr == "tcp://sentry-service-3.test-namespace:1234" {
			hasSentry3 = true
		}
	}
	assert.True(t, hasSentry3)

	// Safely clean up
	safeCleanup(sw)
}

func TestReconcileSentries_MultiNamespaceWithCSVSelector(t *testing.T) {
	// Set a timeout for the test to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create SentryWatcher with mocked dependencies (all=true to get sentries from all nodes)
	sw, _, _, clientset := setupMockSentryWatcher(t, true, true, []string{})

	// Create multiple namespaces
	namespaces := []string{"test-namespace-1", "test-namespace-2", "test-namespace-3"}
	for _, ns := range namespaces {
		_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	// Create pods and services in different namespaces with different label combinations
	createPodAndService(t, clientset, "sentry-1", "test-namespace-1", "test-node", map[string]string{
		"app.kubernetes.io/component": "cosmos-sentry",
		"chain":                       "osmosis",
		"network":                     "mainnet",
		"tier":                        "validator", // extra label not in selector
		"region":                      "us-west",   // extra label not in selector
	})

	createPodAndService(t, clientset, "sentry-2", "test-namespace-2", "other-node", map[string]string{
		"app.kubernetes.io/component": "cosmos-sentry",
		"chain":                       "osmosis",
		"network":                     "testnet", // doesn't match selector
		"tier":                        "validator",
	})

	createPodAndService(t, clientset, "sentry-3", "test-namespace-3", "test-node", map[string]string{
		"app.kubernetes.io/component": "cosmos-sentry",
		"chain":                       "osmosis",
		"network":                     "mainnet",
		"role":                        "archive", // extra label not in selector
	})

	createPodAndService(t, clientset, "sentry-4", "test-namespace-3", "test-node", map[string]string{
		"app.kubernetes.io/component": "cosmos-sentry",
		"chain":                       "juno", // doesn't match selector
		"network":                     "mainnet",
		"tier":                        "validator",
	})

	// Create a CSV label string that should match sentry-1 and sentry-3
	sw.labels = "chain=osmosis,network=mainnet"

	// Run reconcileSentries with the selector
	err := sw.reconcileSentries(ctx, 1024)
	require.NoError(t, err)

	// Should have connected to sentry-1 and sentry-3
	assert.Len(t, sw.sentries, 2)

	// Check if the correct sentries are connected
	expectedSentries := map[string]bool{
		"tcp://sentry-1-service.test-namespace-1:1234": true,
		"tcp://sentry-3-service.test-namespace-3:1234": true,
	}

	for addr := range sw.sentries {
		assert.True(t, expectedSentries[addr], "Unexpected sentry connected: %s", addr)
		delete(expectedSentries, addr)
	}

	// Make sure all expected sentries were found
	assert.Len(t, expectedSentries, 0, "Not all expected sentries were connected: %v", expectedSentries)

	// Verify that sentry-2 and sentry-4 are not connected
	for addr := range sw.sentries {
		assert.NotEqual(t, "tcp://sentry-2-service.test-namespace-2:1234", addr)
		assert.NotEqual(t, "tcp://sentry-4-service.test-namespace-3:1234", addr)
	}

	// Safely clean up
	safeCleanup(sw)
}

// Helper function to create a pod and service with the same labels
func createPodAndService(t *testing.T, clientset *fake.Clientset, name, namespace, nodeName string, labels map[string]string) {
	// Create pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-pod",
			Namespace: namespace,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
		},
	}

	_, err := clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create service with the specified labels
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-service",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []corev1.ServicePort{
				{
					Name: "sentry-privval",
					Port: 1234,
				},
			},
		},
	}

	_, err = clientset.CoreV1().Services(namespace).Create(context.Background(), service, metav1.CreateOptions{})
	require.NoError(t, err)
}
