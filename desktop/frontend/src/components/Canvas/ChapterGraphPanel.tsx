// 章节事件图谱面板组件（参考Toonflow）
import { useState, useCallback, useMemo } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';
import type { 
  Chapter, 
  ChapterEvent, 
  CharacterInfo, 
  LocationInfo,
} from '../../types/canvas';

interface ChapterGraphPanelProps {
  onClose: () => void;
}

export function ChapterGraphPanel({ onClose }: ChapterGraphPanelProps) {
  const [activeTab, setActiveTab] = useState<'chapters' | 'events' | 'characters' | 'locations'>('chapters');
  const [selectedChapterId, setSelectedChapterId] = useState<string | null>(null);
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);

  const {
    chapterGraph,
    addChapter,
    updateChapter,
    addEvent,
    updateEvent,
    addCharacter,
    addLocation,
  } = useCanvasStore();

  // 获取当前选中的章节
  const selectedChapter = useMemo(() => {
    if (!chapterGraph || !selectedChapterId) return null;
    return chapterGraph.chapters.find(c => c.id === selectedChapterId);
  }, [chapterGraph, selectedChapterId]);

  // 获取当前选中的事件
  const selectedEvent = useMemo(() => {
    if (!chapterGraph || !selectedEventId) return null;
    return chapterGraph.events.find(e => e.id === selectedEventId);
  }, [chapterGraph, selectedEventId]);

  // 获取章节的事件
  const chapterEvents = useMemo(() => {
    if (!chapterGraph || !selectedChapterId) return [];
    return chapterGraph.events.filter(e => e.chapterId === selectedChapterId);
  }, [chapterGraph, selectedChapterId]);

  // 添加新章节
  const handleAddChapter = useCallback(() => {
    if (!chapterGraph) return;
    
    const newChapter: Chapter = {
      id: `chapter-${Date.now()}`,
      title: `Chapter ${chapterGraph.chapters.length + 1}`,
      summary: '',
      events: [],
      characters: [],
      locations: [],
    };
    
    addChapter(newChapter);
    setSelectedChapterId(newChapter.id);
  }, [chapterGraph, addChapter]);

  // 添加新事件
  const handleAddEvent = useCallback(() => {
    if (!chapterGraph || !selectedChapterId) return;
    
    const newEvent: ChapterEvent = {
      id: `event-${Date.now()}`,
      chapterId: selectedChapterId,
      eventId: `evt-${Date.now()}`,
      description: 'New event',
      characters: [],
      locations: [],
      time: '',
      importance: 'medium',
      dependencies: [],
    };
    
    addEvent(newEvent);
    setSelectedEventId(newEvent.id);
  }, [chapterGraph, selectedChapterId, addEvent]);

  // 添加新角色
  const handleAddCharacter = useCallback(() => {
    if (!chapterGraph) return;
    
    const newCharacter: CharacterInfo = {
      id: `char-${Date.now()}`,
      name: 'New Character',
      description: '',
      aliases: [],
      relationships: [],
      appearances: [],
    };
    
    addCharacter(newCharacter);
  }, [chapterGraph, addCharacter]);

  // 添加新场景
  const handleAddLocation = useCallback(() => {
    if (!chapterGraph) return;
    
    const newLocation: LocationInfo = {
      id: `loc-${Date.now()}`,
      name: 'New Location',
      description: '',
      aliases: [],
      appearances: [],
    };
    
    addLocation(newLocation);
  }, [chapterGraph, addLocation]);

  // 渲染章节列表
  const renderChapters = () => {
    if (!chapterGraph) {
      return (
        <div className="chapter-graph-panel__empty">
          <p>No chapter graph created yet.</p>
          <button 
            className="chapter-graph-panel__btn"
            onClick={() => {/* 创建新图谱 */}}
          >
            Create Chapter Graph
          </button>
        </div>
      );
    }

    return (
      <div className="chapter-graph-panel__list">
        {chapterGraph.chapters.map(chapter => (
          <div
            key={chapter.id}
            className={`chapter-graph-panel__item ${selectedChapterId === chapter.id ? 'selected' : ''}`}
            onClick={() => setSelectedChapterId(chapter.id)}
          >
            <div className="chapter-graph-panel__item-title">{chapter.title}</div>
            <div className="chapter-graph-panel__item-meta">
              {chapter.events.length} events · {chapter.characters.length} characters
            </div>
          </div>
        ))}
        <button 
          className="chapter-graph-panel__btn chapter-graph-panel__btn--add"
          onClick={handleAddChapter}
        >
          + Add Chapter
        </button>
      </div>
    );
  };

  // 渲染事件列表
  const renderEvents = () => {
    if (!chapterGraph || !selectedChapterId) {
      return (
        <div className="chapter-graph-panel__empty">
          <p>Select a chapter to view events</p>
        </div>
      );
    }

    return (
      <div className="chapter-graph-panel__list">
        {chapterEvents.map(event => (
          <div
            key={event.id}
            className={`chapter-graph-panel__item ${selectedEventId === event.id ? 'selected' : ''}`}
            onClick={() => setSelectedEventId(event.id)}
          >
            <div className="chapter-graph-panel__item-header">
              <span className={`chapter-graph-panel__importance chapter-graph-panel__importance--${event.importance}`}>
                {event.importance}
              </span>
              <div className="chapter-graph-panel__item-title">{event.description}</div>
            </div>
            <div className="chapter-graph-panel__item-meta">
              {event.characters.length} characters · {event.locations.length} locations
            </div>
          </div>
        ))}
        <button 
          className="chapter-graph-panel__btn chapter-graph-panel__btn--add"
          onClick={handleAddEvent}
        >
          + Add Event
        </button>
      </div>
    );
  };

  // 渲染角色列表
  const renderCharacters = () => {
    if (!chapterGraph) return null;

    return (
      <div className="chapter-graph-panel__list">
        {chapterGraph.characters.map(character => (
          <div
            key={character.id}
            className="chapter-graph-panel__item"
          >
            <div className="chapter-graph-panel__item-title">{character.name}</div>
            <div className="chapter-graph-panel__item-meta">
              {character.appearances.length} chapters · {character.relationships.length} relationships
            </div>
            {character.description && (
              <div className="chapter-graph-panel__item-desc">{character.description}</div>
            )}
          </div>
        ))}
        <button 
          className="chapter-graph-panel__btn chapter-graph-panel__btn--add"
          onClick={handleAddCharacter}
        >
          + Add Character
        </button>
      </div>
    );
  };

  // 渲染场景列表
  const renderLocations = () => {
    if (!chapterGraph) return null;

    return (
      <div className="chapter-graph-panel__list">
        {chapterGraph.locations.map(location => (
          <div
            key={location.id}
            className="chapter-graph-panel__item"
          >
            <div className="chapter-graph-panel__item-title">{location.name}</div>
            <div className="chapter-graph-panel__item-meta">
              {location.appearances.length} chapters
            </div>
            {location.description && (
              <div className="chapter-graph-panel__item-desc">{location.description}</div>
            )}
          </div>
        ))}
        <button 
          className="chapter-graph-panel__btn chapter-graph-panel__btn--add"
          onClick={handleAddLocation}
        >
          + Add Location
        </button>
      </div>
    );
  };

  // 渲染详情面板
  const renderDetail = () => {
    if (activeTab === 'chapters' && selectedChapter) {
      return (
        <div className="chapter-graph-panel__detail">
          <h3>{selectedChapter.title}</h3>
          <div className="chapter-graph-panel__detail-section">
            <label>Summary</label>
            <textarea
              value={selectedChapter.summary}
              onChange={(e) => updateChapter(selectedChapterId!, { summary: e.target.value })}
              placeholder="Chapter summary..."
            />
          </div>
          <div className="chapter-graph-panel__detail-section">
            <label>Events ({chapterEvents.length})</label>
            <div className="chapter-graph-panel__detail-list">
              {chapterEvents.map(event => (
                <div key={event.id} className="chapter-graph-panel__detail-item">
                  <span className={`chapter-graph-panel__importance chapter-graph-panel__importance--${event.importance}`}>
                    {event.importance}
                  </span>
                  <span>{event.description}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      );
    }

    if (activeTab === 'events' && selectedEvent) {
      return (
        <div className="chapter-graph-panel__detail">
          <h3>Event Details</h3>
          <div className="chapter-graph-panel__detail-section">
            <label>Description</label>
            <textarea
              value={selectedEvent.description}
              onChange={(e) => updateEvent(selectedEventId!, { description: e.target.value })}
              placeholder="Event description..."
            />
          </div>
          <div className="chapter-graph-panel__detail-section">
            <label>Time</label>
            <input
              type="text"
              value={selectedEvent.time}
              onChange={(e) => updateEvent(selectedEventId!, { time: e.target.value })}
              placeholder="When does this happen?"
            />
          </div>
          <div className="chapter-graph-panel__detail-section">
            <label>Importance</label>
            <select
              value={selectedEvent.importance}
              onChange={(e) => updateEvent(selectedEventId!, { importance: e.target.value as any })}
            >
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
          </div>
        </div>
      );
    }

    return null;
  };

  return (
    <div className="chapter-graph-panel">
      <div className="chapter-graph-panel__header">
        <div className="chapter-graph-panel__title">Chapter Graph</div>
        <button className="chapter-graph-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      <div className="chapter-graph-panel__tabs">
        <button
          className={`chapter-graph-panel__tab ${activeTab === 'chapters' ? 'active' : ''}`}
          onClick={() => setActiveTab('chapters')}
        >
          Chapters
        </button>
        <button
          className={`chapter-graph-panel__tab ${activeTab === 'events' ? 'active' : ''}`}
          onClick={() => setActiveTab('events')}
        >
          Events
        </button>
        <button
          className={`chapter-graph-panel__tab ${activeTab === 'characters' ? 'active' : ''}`}
          onClick={() => setActiveTab('characters')}
        >
          Characters
        </button>
        <button
          className={`chapter-graph-panel__tab ${activeTab === 'locations' ? 'active' : ''}`}
          onClick={() => setActiveTab('locations')}
        >
          Locations
        </button>
      </div>

      <div className="chapter-graph-panel__content">
        <div className="chapter-graph-panel__list-container">
          {activeTab === 'chapters' && renderChapters()}
          {activeTab === 'events' && renderEvents()}
          {activeTab === 'characters' && renderCharacters()}
          {activeTab === 'locations' && renderLocations()}
        </div>

        <div className="chapter-graph-panel__detail-container">
          {renderDetail()}
        </div>
      </div>

      <div className="chapter-graph-panel__footer">
        <div className="chapter-graph-panel__stats">
          {chapterGraph ? (
            <>
              {chapterGraph.chapters.length} chapters · {chapterGraph.events.length} events · {chapterGraph.characters.length} characters
            </>
          ) : (
            'No chapter graph'
          )}
        </div>
      </div>
    </div>
  );
}
