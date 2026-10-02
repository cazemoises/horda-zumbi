import { useEffect, useRef, useState } from 'react';
import './App.css';
import facaSprite from './assets/sprites/faca_walk.png';
import lanternaSprite from './assets/sprites/lanterna_walk.png';
import zombie1Attack from './assets/sprites/Zombie_1/Attack.png';
import zombie1Dead from './assets/sprites/Zombie_1/Dead.png';
import zombie1Hurt from './assets/sprites/Zombie_1/Hurt.png';
import zombie1Idle from './assets/sprites/Zombie_1/Idle.png';
import zombie1Walk from './assets/sprites/Zombie_1/Walk.png';
import zombie2Attack from './assets/sprites/Zombie_2/Attack.png';
import zombie2Dead from './assets/sprites/Zombie_2/Dead.png';
import zombie2Hurt from './assets/sprites/Zombie_2/Hurt.png';
import zombie2Idle from './assets/sprites/Zombie_2/Idle.png';
import zombie2Walk from './assets/sprites/Zombie_2/Walk.png';
import zombie3Attack from './assets/sprites/Zombie_3/Attack.png';
import zombie3Dead from './assets/sprites/Zombie_3/Dead.png';
import zombie3Hurt from './assets/sprites/Zombie_3/Hurt.png';
import zombie3Idle from './assets/sprites/Zombie_3/Idle.png';
import zombie3Walk from './assets/sprites/Zombie_3/Walk.png';
import zombie4Attack from './assets/sprites/Zombie_4/Attack.png';
import zombie4Dead from './assets/sprites/Zombie_4/Dead.png';
import zombie4Hurt from './assets/sprites/Zombie_4/Hurt.png';
import zombie4Idle from './assets/sprites/Zombie_4/Idle.png';
import zombie4Walk from './assets/sprites/Zombie_4/Walk.png';
import urbanMap from './assets/tiles/urban-kit/map.json';
import urbanTileset from './assets/tiles/urban-kit/Tilemap/tilemap_packed.png';
import type { TileMap } from './assets/tiles/urban-kit/map-types';

type Player = {
  id: string;
  name: string;
  character: string;
  x: number;
  y: number;
  health: number;
};

type Zombie = {
  id: string;
  type: number;
  x: number;
  y: number;
  health: number;
  attacking: boolean;
  hurt: boolean;
  dead: boolean;
};

type Character = 'tank' | 'medic' | 'scout';
type ZombieAnimationName = 'idle' | 'walk' | 'attack' | 'hurt' | 'dead';

type LoadedZombieSprite = {
  image: HTMLImageElement;
  frameWidth: number;
  frameHeight: number;
  frameCount: number;
};

type ZombieSpriteSet = Record<ZombieAnimationName, LoadedZombieSprite | null>;

type LoadedSpriteSheet = {
  image: HTMLImageElement;
  frameWidth: number;
  frameHeight: number;
  rows: number;
};

type Movement = {
  dx: number;
  dy: number;
};

type AnimationState = {
  frame: number;
  spriteRow: number;
  lastX: number;
  lastY: number;
  lastMovementAt: number;
  lastFrameAt: number;
};

type ZombieAnimationState = {
  frame: number;
  animation: ZombieAnimationName;
  directionX: -1 | 1;
  lastX: number;
  lastY: number;
  lastMovementAt: number;
  lastFrameAt: number;
};

const movementByKey: Record<string, Movement> = {
  w: { dx: 0, dy: -1 },
  s: { dx: 0, dy: 1 },
  a: { dx: -1, dy: 0 },
  d: { dx: 1, dy: 0 },
};

const zombieSpriteSources: Record<number, Record<ZombieAnimationName, string>> = {
  1: { idle: zombie1Idle, walk: zombie1Walk, attack: zombie1Attack, hurt: zombie1Hurt, dead: zombie1Dead },
  2: { idle: zombie2Idle, walk: zombie2Walk, attack: zombie2Attack, hurt: zombie2Hurt, dead: zombie2Dead },
  3: { idle: zombie3Idle, walk: zombie3Walk, attack: zombie3Attack, hurt: zombie3Hurt, dead: zombie3Dead },
  4: { idle: zombie4Idle, walk: zombie4Walk, attack: zombie4Attack, hurt: zombie4Hurt, dead: zombie4Dead },
};

const MEDIC_ROW_INSET_PX = 0; // original: 4
const ZOMBIE_FRAME_INTERVAL_MS = 110;
const TILE_SIZE = 16;
const MAP_ORIGIN = -4000;
const TILESET_COLUMNS = 27;
const MAP_WIDTH = 500;
const MAP_HEIGHT = 500;
const tileMap = urbanMap as TileMap;

function App() {
  const [playerName, setPlayerName] = useState('');
  const [selectedCharacter, setSelectedCharacter] = useState<Character>('tank');
  const [isConnected, setIsConnected] = useState(false);
  const [players, setPlayers] = useState<Player[]>([]);
  const [logs, setLogs] = useState<string[]>([]);
  const wsRef = useRef<WebSocket | null>(null);
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const spriteSheetsRef = useRef<Record<Character, LoadedSpriteSheet | null>>({
    tank: null,
    medic: null,
    scout: null,
  });
  const fallbackSpriteSheetRef = useRef<LoadedSpriteSheet | null>(null);
  const playersRef = useRef<Player[]>([]);
  const zombiesRef = useRef<Zombie[]>([]);
  const zombieSpritesRef = useRef<Record<number, ZombieSpriteSet>>({
    1: { idle: null, walk: null, attack: null, hurt: null, dead: null },
    2: { idle: null, walk: null, attack: null, hurt: null, dead: null },
    3: { idle: null, walk: null, attack: null, hurt: null, dead: null },
    4: { idle: null, walk: null, attack: null, hurt: null, dead: null },
  });
  const pressedKeysRef = useRef<Set<string>>(new Set());
  const attackFlashUntilRef = useRef<Record<string, number>>({});
  const localPlayerIdRef = useRef<string | null>(null);
  const animationStateRef = useRef<Record<string, AnimationState>>({});
  const zombieAnimationStateRef = useRef<Record<string, ZombieAnimationState>>({});
  const medicInsetRowRef = useRef<Record<string, number>>({});
  const localPlayerDrawLoggedRef = useRef(false);
  const tilemapImageRef = useRef<HTMLImageElement | null>(null);

  const updatePlayers = (nextPlayers: Player[]) => {
    playersRef.current = nextPlayers;
    setPlayers(nextPlayers);
  };

  const addLog = (msg: string) => {
    setLogs((prev) => [...prev, msg]);
  };

  const sendMovement = () => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return;

    const movement = [...pressedKeysRef.current].reduce(
      (current, key) => ({
        dx: current.dx + (movementByKey[key]?.dx ?? 0),
        dy: current.dy + (movementByKey[key]?.dy ?? 0),
      }),
      { dx: 0, dy: 0 },
    );

    ws.send(JSON.stringify({
      type: 'move',
      dx: Math.max(-1, Math.min(1, movement.dx)),
      dy: Math.max(-1, Math.min(1, movement.dy)),
    }));
  };

  const sendAttack = (source: 'space' | 'click' | 'button') => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return;

    const localPlayer = playersRef.current.find((player) => player.id === localPlayerIdRef.current);
    const payload = { type: 'attack' };
    console.log('Ataque disparado pelo cliente', {
      source,
      position: localPlayer ? { x: localPlayer.x, y: localPlayer.y } : null,
    });
    ws.send(JSON.stringify(payload));
    console.log('Payload de ataque enviado', {
      payload: JSON.stringify(payload),
      position: localPlayer ? { x: localPlayer.x, y: localPlayer.y } : null,
    });
    if (localPlayerIdRef.current) {
      attackFlashUntilRef.current[localPlayerIdRef.current] = performance.now() + 140;
    }
  };

  const connect = () => {
    if (!playerName.trim()) return;

    const ws = new WebSocket('ws://127.0.0.1:8080/ws/rooms/lobby-1');
    wsRef.current = ws;

    ws.onopen = () => {
      addLog(`Conectado! Enviando JOIN como "${playerName}"...`);
      ws.send(JSON.stringify({ type: 'join', name: playerName, character: selectedCharacter }));
    };

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      addLog(`Recebido: ${JSON.stringify(data)}`);

      switch (data.type) {
        case 'joined':
          setIsConnected(true);
          localPlayerIdRef.current = data.player_id;
          updatePlayers(data.room.players);
          break;
        case 'player_joined':
          if (data.player) {
            updatePlayers([...playersRef.current, data.player]);
          }
          break;
        case 'player_left':
          updatePlayers(playersRef.current.filter((player) => player.id !== data.player_id));
          break;
        case 'state_update':
          updatePlayers(data.players);
          zombiesRef.current = data.zombies || [];
          break;
        case 'error':
          addLog(`Rejeitado pelo servidor: ${data.message || 'Erro desconhecido'}`);
          ws.close();
          break;
      }
    };

    ws.onclose = () => {
      addLog('Conexão fechada');
      setIsConnected(false);
      pressedKeysRef.current.clear();
      updatePlayers([]);
      zombiesRef.current = [];
      localPlayerIdRef.current = null;
      wsRef.current = null;
    };
  };

  const disconnect = () => {
    wsRef.current?.close();
  };

  const handleCanvasClick = () => {
    console.log('attack key/click detected');
    sendAttack('click');
  };

  const enterFullscreen = () => {
    void canvasRef.current?.requestFullscreen();
  };

  useEffect(() => {
    const sources: Record<Character, string> = {
      tank: lanternaSprite,
      medic: facaSprite,
      scout: '',
    };

    (Object.keys(sources) as Character[]).forEach((character) => {
      if (!sources[character]) return;
      const image = new Image();
      image.src = sources[character];
      image.onload = () => {
        console.log('SpriteSheet carregada', {
          character,
          file: sources[character],
          naturalWidth: image.naturalWidth,
          naturalHeight: image.naturalHeight,
          frameWidth: image.naturalWidth / 4,
          frameHeight: image.naturalHeight / 4,
        });
        spriteSheetsRef.current[character] = {
          image,
          frameWidth: image.naturalWidth / 4,
          frameHeight: image.naturalHeight / 4,
          rows: 4,
        };
      };
    });

    const fallback = new Image();
    fallback.src = '/spritesheet.jpeg';
    fallback.onload = () => {
      fallbackSpriteSheetRef.current = {
        image: fallback,
        frameWidth: fallback.naturalWidth / 4,
        frameHeight: fallback.naturalHeight / 3,
        rows: 3,
      };
    };
  }, []);

  useEffect(() => {
    const image = new Image();
    image.src = urbanTileset;
    image.onload = () => {
      tilemapImageRef.current = image;
    };
  }, []);

  useEffect(() => {
    Object.entries(zombieSpriteSources).forEach(([type, animations]) => {
      Object.entries(animations).forEach(([animation, source]) => {
        const image = new Image();
        image.src = source;
        image.onload = () => {
          const frameCount = image.naturalWidth / 128;
          zombieSpritesRef.current[Number(type)][animation as ZombieAnimationName] = {
            image,
            frameWidth: 128,
            frameHeight: 128,
            frameCount,
          };
        };
      });
    });
  }, []);

  useEffect(() => {
    const handleAttackKeyDown = (event: KeyboardEvent) => {
      console.log('attack key/click detected');
      if (!isConnected || event.code !== 'Space') return;

      event.preventDefault();
      sendAttack('space');
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      if (!isConnected || !movementByKey[key]) return;

      event.preventDefault();
      pressedKeysRef.current.add(key);
      sendMovement();
    };

    const handleKeyUp = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      if (!movementByKey[key]) return;

      pressedKeysRef.current.delete(key);
      if (!isConnected) return;

      event.preventDefault();
      sendMovement();
    };

    window.addEventListener('keydown', handleAttackKeyDown);
    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keyup', handleKeyUp);
    const movementInterval = window.setInterval(() => {
      if (isConnected && pressedKeysRef.current.size > 0) {
        sendMovement();
      }
    }, 33);

    return () => {
      window.removeEventListener('keydown', handleAttackKeyDown);
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('keyup', handleKeyUp);
      window.clearInterval(movementInterval);
    };
  }, [isConnected]);

  useEffect(() => {
    let animationFrame = 0;

    const render = () => {
      const canvas = canvasRef.current;
      const context = canvas?.getContext('2d');
      if (canvas && context) {
        const now = performance.now();
        context.clearRect(0, 0, canvas.width, canvas.height);
        context.textAlign = 'center';
        context.font = '14px sans-serif';

        const localPlayer = playersRef.current.find((player) => player.id === localPlayerIdRef.current);
        const cameraX = (localPlayer?.x ?? 0) - canvas.width / 2;
        const cameraY = (localPlayer?.y ?? 0) - canvas.height / 2;
        const tilemapImage = tilemapImageRef.current;
        if (tilemapImage?.complete && tilemapImage.naturalWidth > 0) {
          const firstColumn = Math.max(0, Math.floor((cameraX - MAP_ORIGIN) / TILE_SIZE) - 1);
          const lastColumn = Math.min(
            MAP_WIDTH - 1,
            Math.ceil((cameraX + canvas.width - MAP_ORIGIN) / TILE_SIZE) + 1,
          );
          const firstRow = Math.max(0, Math.floor((cameraY - MAP_ORIGIN) / TILE_SIZE) - 1);
          const lastRow = Math.min(
            MAP_HEIGHT - 1,
            Math.ceil((cameraY + canvas.height - MAP_ORIGIN) / TILE_SIZE) + 1,
          );
          tileMap.layers
            .filter((layer) => layer.name === 'chao' || layer.name === 'predios')
            .forEach((layer) => {
              for (let row = firstRow; row <= lastRow; row += 1) {
                for (let column = firstColumn; column <= lastColumn; column += 1) {
                  const gid = layer.data[row * MAP_WIDTH + column];
                  if (gid === 0) continue;
                  const tileIndex = gid - 1;
                  context.drawImage(
                    tilemapImage,
                    (tileIndex % TILESET_COLUMNS) * TILE_SIZE,
                    Math.floor(tileIndex / TILESET_COLUMNS) * TILE_SIZE,
                    TILE_SIZE,
                    TILE_SIZE,
                    column * TILE_SIZE + MAP_ORIGIN - cameraX,
                    row * TILE_SIZE + MAP_ORIGIN - cameraY,
                    TILE_SIZE,
                    TILE_SIZE,
                  );
                }
              }
            });
        }

        playersRef.current.forEach((player) => {
          const spriteSheet = spriteSheetsRef.current[player.character as Character]
            ?? fallbackSpriteSheetRef.current;
          const image = spriteSheet?.image;
          const animation = animationStateRef.current[player.id] ?? {
            frame: 0,
            spriteRow: 0,
            lastX: player.x,
            lastY: player.y,
            lastMovementAt: 0,
            lastFrameAt: now,
          };
          const movement = {
            dx: player.x - animation.lastX,
            dy: player.y - animation.lastY,
          };
          if (movement.dx !== 0 || movement.dy !== 0) {
            if (Math.abs(movement.dy) >= Math.abs(movement.dx) && movement.dy < 0) {
              animation.spriteRow = 1;
            } else if (Math.abs(movement.dy) < Math.abs(movement.dx) && movement.dx < 0) {
              animation.spriteRow = 2;
            } else if (Math.abs(movement.dy) < Math.abs(movement.dx) && movement.dx > 0) {
              animation.spriteRow = 3;
            } else {
              animation.spriteRow = 0;
            }
            animation.lastMovementAt = now;
          }
          const spriteRow = animation.spriteRow;
          const isMoving = now - animation.lastMovementAt < 180;
          if (!isMoving) {
            animation.frame = 0;
            animation.lastFrameAt = now;
          } else if (now - animation.lastFrameAt >= 140) {
            animation.frame = (animation.frame + 1) % 4;
            animation.lastFrameAt = now;
          }
          animation.lastX = player.x;
          animation.lastY = player.y;
          animationStateRef.current[player.id] = animation;

          if (spriteSheet && image && image.complete && image.naturalWidth > 0) {
            const sWidth = spriteSheet.frameWidth;
            const sHeight = spriteSheet.frameHeight;
            const rowInset = player.character === 'medic' ? MEDIC_ROW_INSET_PX : 0;
            const sourceY = spriteRow * sHeight + rowInset;
            const sourceHeight = sHeight - rowInset * 2;
            const previousSpriteRow = medicInsetRowRef.current[player.id];
            if (
              player.character === 'medic'
              && player.id === localPlayerIdRef.current
              && previousSpriteRow !== spriteRow
            ) {
              console.log('Medic sprite recorte (mudança de linha)', {
                inset: rowInset,
                spriteRow,
                sy: sourceY,
                sHeight: sourceHeight,
                originalFrameHeight: sHeight,
              });
              medicInsetRowRef.current[player.id] = spriteRow;
            }
            const dWidth = 144;
            const dHeight = 165;
            const frameSequence = player.character === 'tank' && (spriteRow === 2 || spriteRow === 3)
              ? [0, 2, 3, 2]
              : [0, 1, 2, 3];
            const drawX = player.x - cameraX - dWidth / 2;
            const drawY = player.y - cameraY - dHeight / 2;
            if (player.id === localPlayerIdRef.current && !localPlayerDrawLoggedRef.current) {
              console.log('Posição de desenho do jogador local', {
                world: { x: player.x, y: player.y },
                screen: { x: drawX, y: drawY },
                canvas: { width: canvas.width, height: canvas.height },
              });
              localPlayerDrawLoggedRef.current = true;
            }
            context.drawImage(
              image,
              frameSequence[animation.frame] * sWidth,
              sourceY,
              sWidth,
              sourceHeight,
              drawX,
              drawY,
              dWidth,
              dHeight,
            );
          } else {
            context.beginPath();
            context.arc(player.x - cameraX, player.y - cameraY, 20, 0, Math.PI * 2);
            context.fillStyle = '#ffffff';
            context.fill();
          }

          context.fillStyle = '#ffffff';
          context.fillText(player.name, player.x - cameraX, player.y - cameraY - 28);

          if (performance.now() < (attackFlashUntilRef.current[player.id] ?? 0)) {
            context.beginPath();
            context.arc(player.x - cameraX, player.y - cameraY, 60, 0, Math.PI * 2);
            context.strokeStyle = 'rgba(255, 240, 120, 0.85)';
            context.lineWidth = 5;
            context.stroke();
          }
        });

        zombiesRef.current.forEach((zombie) => {
          const animation = zombieAnimationStateRef.current[zombie.id] ?? {
            frame: 0,
            animation: 'idle' as ZombieAnimationName,
            directionX: 1 as const,
            lastX: zombie.x,
            lastY: zombie.y,
            lastMovementAt: 0,
            lastFrameAt: now,
          };
          const deltaX = zombie.x - animation.lastX;
          const deltaY = zombie.y - animation.lastY;
          if (Math.abs(deltaX) > 0.01 || Math.abs(deltaY) > 0.01) {
            animation.lastMovementAt = now;
            if (Math.abs(deltaX) > 0.01) {
              animation.directionX = deltaX < 0 ? -1 : 1;
            }
          }
          const isMoving = now - animation.lastMovementAt < 180;
          const nextAnimation: ZombieAnimationName = zombie.dead
            ? 'dead'
            : zombie.hurt
              ? 'hurt'
              : zombie.attacking
                ? 'attack'
                : isMoving
                  ? 'walk'
                  : 'idle';
          if (animation.animation !== nextAnimation) {
            animation.animation = nextAnimation;
            animation.frame = 0;
            animation.lastFrameAt = now;
          }
          const sprite = zombieSpritesRef.current[zombie.type]?.[animation.animation];
          if (sprite && now - animation.lastFrameAt >= ZOMBIE_FRAME_INTERVAL_MS) {
            if (animation.animation === 'dead') {
              animation.frame = Math.min(animation.frame + 1, sprite.frameCount - 1);
            } else {
              animation.frame = (animation.frame + 1) % sprite.frameCount;
            }
            animation.lastFrameAt = now;
          }
          animation.lastX = zombie.x;
          animation.lastY = zombie.y;
          zombieAnimationStateRef.current[zombie.id] = animation;

          if (sprite && sprite.image.complete && sprite.image.naturalWidth > 0) {
            const dWidth = 288;
            const dHeight = 288;
            context.save();
            context.translate(zombie.x - cameraX, zombie.y - cameraY);
            context.scale(animation.directionX, 1);
            context.drawImage(
              sprite.image,
              animation.frame * sprite.frameWidth,
              0,
              sprite.frameWidth,
              sprite.frameHeight,
              -dWidth / 2,
              -dHeight / 2,
              dWidth,
              dHeight,
            );
            context.restore();
          }
        });
      }

      animationFrame = requestAnimationFrame(render);
    };

    animationFrame = requestAnimationFrame(render);
    return () => cancelAnimationFrame(animationFrame);
  }, []);

  useEffect(() => () => {
    wsRef.current?.close();
  }, []);

  // Força o foco no canvas assim que a conexão for estabelecida
  useEffect(() => {
    if (isConnected && canvasRef.current) {
      canvasRef.current.focus();
    }
  }, [isConnected]);

  return (
    <div style={{ padding: '2rem', textAlign: 'left' }}>
      <h1>Horda Zumbi - Multiplayer</h1>

      {!isConnected ? (
        <div style={{ marginBottom: '20px' }}>
          <input
            type="text"
            placeholder="Digite seu nome"
            value={playerName}
            onChange={(event) => setPlayerName(event.target.value)}
            style={{ padding: '8px', marginRight: '10px' }}
          />
          <div style={{ margin: '12px 0' }}>
            <p>Escolha sua classe:</p>
            {(['tank', 'medic', 'scout'] as Character[]).map((character) => (
              <button
                key={character}
                type="button"
                onClick={() => setSelectedCharacter(character)}
                style={{
                  padding: '8px 14px',
                  marginRight: '8px',
                  border: selectedCharacter === character ? '2px solid #fff' : '1px solid #555',
                  background: selectedCharacter === character ? '#444' : '#222',
                  color: '#fff',
                }}
              >
                {character === 'tank' ? 'Tank (Lanterna)' : character === 'medic' ? 'Medic (Faca)' : 'Scout (Taco)'}
              </button>
            ))}
          </div>
          <button onClick={connect} style={{ padding: '8px 16px' }}>Entrar na Sala</button>
        </div>
      ) : (
        <div style={{ marginBottom: '20px', padding: '1rem', border: '1px solid #444', borderRadius: '8px' }}>
          <p>Logado como: <strong>{playerName}</strong></p>
          <button onClick={disconnect} style={{ padding: '8px 16px', backgroundColor: '#d32f2f' }}>Sair da Sala</button>
          <button onClick={enterFullscreen} style={{ padding: '8px 16px', marginLeft: '8px' }}>Tela Cheia</button>
          <button onClick={() => sendAttack('button')} style={{ padding: '8px 16px', marginLeft: '8px' }}>Atacar (Espaço)</button>
          <p>Jogadores na sala: {players.length} (use WASD para mover)</p>
          <canvas
            ref={canvasRef}
            tabIndex={0}
            className="game-canvas"
            onClick={handleCanvasClick}
            width={1200}
            height={800}
            style={{
              border: '1px solid #555',
              background: '#222',
              display: 'block',
              margin: '0 auto',
              outline: 'none',
              width: '100%',
              height: 'auto',
              maxWidth: '1200px',
            }}
          />
        </div>
      )}

      <h3>Logs do Servidor:</h3>
      <div style={{
        background: '#1e1e1e',
        color: '#4af626',
        padding: '1rem',
        height: '300px',
        overflowY: 'auto',
        fontFamily: 'monospace',
        borderRadius: '8px',
      }}>
        {logs.map((log, index) => <div key={index} style={{ marginBottom: '4px' }}>{log}</div>)}
      </div>
    </div>
  );
}

export default App;
