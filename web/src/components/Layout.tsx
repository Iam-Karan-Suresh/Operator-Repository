import React from 'react';
import { Sidebar } from './Sidebar';
import Footer from './Footer';

interface LayoutProps {
  children: React.ReactNode;
  currentView: string;
  onSelectView: (v: string) => void;
  uiSettings: {
    name: string;
    profession: string;
    team: string;
  };
}

export function Layout({ children, currentView, onSelectView, uiSettings }: LayoutProps) {
  return (
    <div className="flex h-screen bg-background text-foreground overflow-hidden selection:bg-primary/30">
      <Sidebar currentView={currentView} onSelectView={onSelectView} uiSettings={uiSettings} />
      <main className="flex-1 flex flex-col overflow-hidden relative">
        {/* Subtle background glow effect */}
        <div className="absolute top-0 inset-x-0 h-[500px] bg-gradient-to-b from-primary/5 to-transparent pointer-events-none" />
        
        <div className="relative z-10 w-full flex-1 overflow-y-auto">
          {children}
        </div>
        
        <Footer 
          name={uiSettings.name} 
          profession={uiSettings.profession} 
          team={uiSettings.team} 
        />
      </main>
    </div>
  );
}
