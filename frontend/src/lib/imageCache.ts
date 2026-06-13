class ImageCacheStore {

  private loaded = new Set<string>();
  private pending = new Map<string, Promise<void>>();

  isLoaded(src: string): boolean {

    return this.loaded.has(src);

  }

  preload(src: string): Promise<void> {

    if (!src) {

      return Promise.resolve();

    }

    if (this.loaded.has(src)) {

      return Promise.resolve();

    }

    const existing = this.pending.get(src);

    if (existing) {

      return existing;

    }

    const promise = new Promise<void>((resolve, reject) => {

      const img = new Image();

      img.decoding = "async";

      img.onload = () => {

        this.loaded.add(src);

        resolve();

      };

      img.onerror = () => reject(new Error("image load failed"));

      img.src = src;

    }).finally(() => {

      this.pending.delete(src);

    });

    this.pending.set(src, promise);

    return promise;

  }

}

export const imageCache = new ImageCacheStore();