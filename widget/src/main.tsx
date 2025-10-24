import { render } from 'preact';
import { CommentWidget } from './components/CommentWidget';
import type { OrbitHallInitConfig } from './types';
import { I18nProvider } from './i18n/context';
import type { Locale } from './i18n/locales';
import './styles.css';

// IIFE로 전역 스코프 오염 방지
(function() {
  // 환경별 API URL (빌드 시점에 치환됨)
  const API_URL = process.env.ORB_PUBLIC_API_URL || '';

  const observers = new Map<string, MutationObserver>();
  let globalApiKey = '';
  let globalLocale: Locale = 'ko';
  let documentObserver: MutationObserver | null = null;
  let isInitialized = false;

  // 고유 해시 생성 함수
  function generateHash(element: HTMLElement): string {
    const widgetType = element.getAttribute('data-widget-type') || 'widget';
    const postSlug = element.getAttribute('data-post-slug') || '';
    const timestamp = Date.now();
    const random = Math.random().toString(36).substring(2, 7);

    // 간단한 해시 생성 (base64 인코딩 + 랜덤)
    const baseString = `${widgetType}-${postSlug}-${timestamp}`;
    const hash = btoa(baseString).replace(/[^a-zA-Z0-9]/g, '').substring(0, 8);

    return `orb-${widgetType}-${hash}-${random}`;
  }

  // 위젯 렌더링 함수
  function renderWidget(container: HTMLElement, apiUrl: string, apiKey: string, locale: Locale) {
    const widgetType = container.getAttribute('data-widget-type');
    const postSlug = container.getAttribute('data-post-slug');

    if (!postSlug) {
      console.error('OrbitHall: data-post-slug attribute is required');
      return;
    }

    // 위젯 타입별 렌더링
    switch (widgetType) {
      case 'comments':
        render(
          <I18nProvider locale={locale}>
            <CommentWidget apiUrl={apiUrl} apiKey={apiKey} postSlug={postSlug} />
          </I18nProvider>,
          container
        );
        break;

      case 'reactions':
        // 향후 구현
        console.warn('OrbitHall: reactions widget not implemented yet');
        break;

      default:
        console.error(`OrbitHall: Unknown widget type "${widgetType}"`);
    }
  }

  // MutationObserver 설정
  function observeContainer(container: HTMLElement, apiUrl: string, apiKey: string, locale: Locale) {
    // 이미 초기화된 컨테이너인지 확인
    let orbithallId = container.getAttribute('data-orb-id');

    if (orbithallId && observers.has(orbithallId)) {
      // 이미 초기화됨
      return;
    }

    // 고유 ID 생성 및 설정
    if (!orbithallId) {
      orbithallId = generateHash(container);
      container.setAttribute('data-orb-id', orbithallId);
      container.setAttribute('data-orb-initialized', 'true');
    }

    // 초기 렌더링
    renderWidget(container, apiUrl, apiKey, locale);

    // attribute 변경 감지
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.type === 'attributes' &&
            mutation.attributeName === 'data-post-slug') {
          renderWidget(container, apiUrl, apiKey, locale);
        }
      });
    });

    observer.observe(container, { attributes: true });
    observers.set(orbithallId, observer);
  }

  // 전역 MutationObserver: 새로운 컨테이너가 추가되는 것을 감지
  function startDocumentObserver() {
    if (documentObserver) {
      return; // 이미 실행 중
    }

    documentObserver = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        mutation.addedNodes.forEach((node) => {
          if (node instanceof HTMLElement) {
            // 추가된 노드가 위젯 컨테이너인 경우
            if (node.hasAttribute('data-orb-container')) {
              observeContainer(node, API_URL, globalApiKey, globalLocale);
            }
            // 추가된 노드의 자식 중에 위젯 컨테이너가 있는 경우
            const containers = node.querySelectorAll('[data-orb-container]');
            containers.forEach((container) => {
              if (container instanceof HTMLElement) {
                observeContainer(container, API_URL, globalApiKey, globalLocale);
              }
            });
          }
        });
      });
    });

    documentObserver.observe(document.body, {
      childList: true,
      subtree: true
    });
  }

  // OrbitHall 네임스페이스
  window.OrbitHall = {
    // 위젯 초기화 함수
    init: function(config: OrbitHallInitConfig) {
      if (isInitialized) {
        console.warn('OrbitHall: already initialized');
        return;
      }

      if (!config.apiKey) {
        console.error('OrbitHall: apiKey is required');
        return;
      }

      globalApiKey = config.apiKey;
      globalLocale = (config.locale || 'ko') as Locale;

      // 기존 컨테이너 초기화
      const containers = document.querySelectorAll('[data-orb-container]');
      containers.forEach((container) => {
        if (container instanceof HTMLElement) {
          observeContainer(container, API_URL, globalApiKey, globalLocale);
        }
      });

      // 전역 observer 시작 (SPA 라우팅 대응)
      startDocumentObserver();

      isInitialized = true;
    },

    destroy: function() {
      observers.forEach((observer) => observer.disconnect());
      observers.clear();
      if (documentObserver) {
        documentObserver.disconnect();
        documentObserver = null;
      }
      isInitialized = false;
    }
  };
})();
