import { describe, expect, it } from 'vitest';

import { buildListQuery, withQuery } from '@/api/client/list';

describe('buildListQuery', () => {
  it('flattens filters, sort, and pagination into query parameters', () => {
    const query = {
      filters: {
        status: 'pending',
        active: true,
        user: 3,
      },
      sort: '-createdAt',
      limit: 25,
      offset: 50,
    };

    const params = buildListQuery(query);

    expect(params).toEqual({
      'filter[status]': 'pending',
      'filter[active]': true,
      'filter[user]': 3,
      sort: '-createdAt',
      'page[limit]': 25,
      'page[offset]': 50,
    });
  });

  it('omits unset and empty filters', () => {
    const query = {
      filters: {
        q: '',
        status: undefined,
        active: false,
      },
    };

    const params = buildListQuery(query);

    expect(params).toEqual({
      'filter[active]': false,
    });
  });

  it('joins a list filter with commas and omits an empty one', () => {
    const query = {
      filters: {
        id: ['a', 'b'],
        status: [],
      },
    };

    const params = buildListQuery(query);

    expect(params).toEqual({
      'filter[id]': 'a,b',
    });
  });

  it('keeps a zero offset', () => {
    const query = {
      offset: 0,
    };

    const params = buildListQuery(query);

    expect(params).toEqual({
      'page[offset]': 0,
    });
  });

  it('returns nothing for an empty query', () => {
    const params = buildListQuery({});

    expect(params).toEqual({});
  });
});

describe('withQuery', () => {
  it('appends the parameters as an encoded query string', () => {
    const params = {
      'filter[name]': 'a b',
      'page[limit]': 10,
      active: true,
    };

    const url = withQuery('/items', params);

    expect(url).toBe('/items?filter%5Bname%5D=a+b&page%5Blimit%5D=10&active=true');
  });

  it('returns the path unchanged without parameters', () => {
    const url = withQuery('/items', {});

    expect(url).toBe('/items');
  });
});
