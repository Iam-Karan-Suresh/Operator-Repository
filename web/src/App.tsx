import { useState, useMemo, useEffect } from 'react';
import { Layout } from './components/Layout';
import { InstanceList } from './components/InstanceList';
import { InstanceDetail } from './components/InstanceDetail';
import { Settings } from './components/Settings';
import { Metrics } from './components/Metrics';
import { useInstances } from './hooks/useInstances';

const API_URL = import.meta.env.VITE_API_URL || '';

function App() {
  const [currentView, setCurrentView] = useState('instances'); // 'instances', 'metrics', 'settings'
  const [selectedInstanceId, setSelectedInstanceId] = useState<string | null>(null);
  const [selectedNamespace, setSelectedNamespace] = useState<string | null>(null);
  const { instances, loading, refetch, refetchInstance } = useInstances();
  
  const [uiSettings, setUiSettings] = useState({
    name: 'User Name',
    profession: 'Project Lead',
    team: 'Cloud Operations'
  });

  useEffect(() => {
    fetch(`${API_URL}/api/settings`)
      .then(res => res.json())
      .then(data => setUiSettings(data))
      .catch(err => console.error("Failed to fetch UI settings", err));
  }, []);

  const handleSelectInstance = (name: string, namespace: string) => {
    setSelectedInstanceId(name);
    setSelectedNamespace(namespace);
  };

  const handleBack = () => {
    setSelectedInstanceId(null);
    setSelectedNamespace(null);
  };

  const selectedInstance = useMemo(() => {
    if (!selectedInstanceId || !selectedNamespace) return null;
    return instances.find(i => i.name === selectedInstanceId && i.namespace === selectedNamespace) || null;
  }, [selectedInstanceId, selectedNamespace, instances]);

  return (
    <Layout currentView={currentView} onSelectView={setCurrentView} uiSettings={uiSettings}>
      <div className="p-8 max-w-7xl mx-auto h-full overflow-y-auto">
        {currentView === 'settings' && <Settings uiSettings={uiSettings} onUpdateSettings={setUiSettings} />}
        {currentView === 'metrics' && <Metrics />}
        {currentView === 'instances' && (
          selectedInstance ? (
            <InstanceDetail 
              instance={selectedInstance} 
              onBack={handleBack} 
              onRefresh={() => refetchInstance(selectedInstance.name, selectedInstance.namespace)} 
              refreshing={loading} 
            />
          ) : (
            <InstanceList onSelectInstance={handleSelectInstance} onRefresh={refetch} refreshing={loading} />
          )
        )}
      </div>
    </Layout>
  );
}

export default App;
