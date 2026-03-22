import React from 'react';

interface FooterProps {
  name: string;
  profession: string;
  team: string;
}

const Footer: React.FC<FooterProps> = ({ name, profession, team }) => {
  return (
    <footer className="mt-auto py-6 px-8 border-t border-white/10 bg-black/20 backdrop-blur-md">
      <div className="flex flex-col md:flex-row justify-between items-center text-sm text-gray-400">
        <div className="flex items-center space-x-2 mb-4 md:mb-0">
          <span className="font-semibold text-gray-300">{name}</span>
          <span className="text-gray-600">|</span>
          <span>{profession}</span>
        </div>
        <div className="flex items-center space-x-4">
          <span className="px-3 py-1 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20">
            {team}
          </span>
          <span>© {new Date().getFullYear()} EC2 Operator</span>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
