import { WinstonMCPTransport, createWinstonTransport } from '../adapters/winston';
import { BunyanMCPStream, createBunyanStream } from '../adapters/bunyan';

// Mock the logger
jest.mock('../logger');

describe('Logging Adapters', () => {
  const mockConfig = {
    serverUrl: 'http://localhost:9080',
    serviceName: 'test-service',
    agentId: 'test-agent'
  };

  describe('Winston Adapter', () => {
    let transport: WinstonMCPTransport;

    beforeEach(() => {
      transport = new WinstonMCPTransport(mockConfig);
    });

    afterEach(async () => {
      await transport.close();
    });

    test('should create winston transport', () => {
      const winstonTransport = createWinstonTransport(mockConfig);
      expect(winstonTransport).toBeInstanceOf(WinstonMCPTransport);
      winstonTransport.close();
    });

    test('should log winston messages', () => {
      const callback = jest.fn();
      const logInfo = {
        level: 'info',
        message: 'Test message',
        userId: '123'
      };

      transport.log(logInfo, callback);
      expect(callback).toHaveBeenCalled();
    });

    test('should map winston levels correctly', () => {
      const callback = jest.fn();
      
      const testCases = [
        { level: 'silly', expectedLevel: 'DEBUG' },
        { level: 'debug', expectedLevel: 'DEBUG' },
        { level: 'info', expectedLevel: 'INFO' },
        { level: 'warn', expectedLevel: 'WARN' },
        { level: 'error', expectedLevel: 'ERROR' },
        { level: 'crit', expectedLevel: 'FATAL' }
      ];

      testCases.forEach(({ level }) => {
        transport.log({ level, message: 'Test' }, callback);
      });

      expect(callback).toHaveBeenCalledTimes(testCases.length);
    });
  });

  describe('Bunyan Adapter', () => {
    let stream: BunyanMCPStream;

    beforeEach(() => {
      stream = new BunyanMCPStream(mockConfig);
    });

    afterEach(async () => {
      await stream.close();
    });

    test('should create bunyan stream', () => {
      const bunyanStream = createBunyanStream(mockConfig);
      expect(bunyanStream).toBeInstanceOf(BunyanMCPStream);
      bunyanStream.close();
    });

    test('should write bunyan records', () => {
      const record = {
        level: 30, // INFO level
        msg: 'Test message',
        time: new Date(),
        userId: '123'
      };

      // Should not throw
      stream.write(record);
    });

    test('should map bunyan levels correctly', () => {
      const testCases = [
        { level: 10, expectedLevel: 'DEBUG' }, // TRACE
        { level: 20, expectedLevel: 'DEBUG' }, // DEBUG
        { level: 30, expectedLevel: 'INFO' },  // INFO
        { level: 40, expectedLevel: 'WARN' },  // WARN
        { level: 50, expectedLevel: 'ERROR' }, // ERROR
        { level: 60, expectedLevel: 'FATAL' }  // FATAL
      ];

      testCases.forEach(({ level }) => {
        stream.write({
          level,
          msg: 'Test message',
          time: new Date()
        });
      });
    });
  });
});