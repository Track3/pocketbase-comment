// @deno-types="https://unpkg.com/pocketbase@0.8.3/dist/pocketbase.es.d.ts"
import PocketBase from "https://unpkg.com/pocketbase@0.8.3/dist/pocketbase.es.mjs";
import { serve } from "https://deno.land/std/http/server.ts";
import { Md5 } from "https://deno.land/std@0.160.0/hash/md5.ts";
import "https://deno.land/std/dotenv/load.ts";

const allowOrigin = Deno.env.get("ALLOW_ORIGIN")?.split(",");
let allowedOrigin = "*";

const pb = new PocketBase(Deno.env.get("PB_URL"));
const _authData = await pb.collection("users").authWithPassword(
  Deno.env.get("PB_USER"),
  Deno.env.get("PB_PASSWORD"),
);

// Validate Url
const isValidUrl = (url) => {
  if (url === "") {
    return true; // "website" filed is optional
  } else {
    try {
      new URL(url);
    } catch (e) {
      console.error(e);
      return false;
    }
    return true;
  }
};

async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  // console.log(req.method, url.pathname, "uri:", url.searchParams.get("uri"));

  const reqestOrigin = req.headers.get("origin");
  if (reqestOrigin === null || !allowOrigin.includes(reqestOrigin)) {
    return new Response("Request is rejected due to CORS policy.");
  } else {
    allowedOrigin = reqestOrigin;
  }

  // List comments for a given page uri
  if (req.method === "GET" && url.searchParams.get("uri") !== null) {
    const resultList = await pb.collection("comments").getFullList(0, {
      filter: `uri='${url.searchParams.get("uri")}'`,
      sort: "created,-parent",
    });

    const commentlist: {
      id: string;
      author: string;
      avatar: string;
      website: string;
      content: string;
      created: string;
      reply: unknown[];
    }[] = [];

    resultList.forEach((item) => {
      if (item.parent === "") {
        commentlist.push({
          id: item.id,
          author: item.author,
          avatar: new Md5().update(item.email).toString(),
          website: item.website,
          content: item.content,
          created: item.created,
          reply: [],
        });
      } else {
        const index = commentlist.findIndex((e) => e.id === item.parent);
        commentlist[index].reply.push({
          id: item.id,
          author: item.author,
          avatar: new Md5().update(item.email).toString(),
          website: item.website,
          content: item.content,
          created: item.created,
        });
      }
    });

    const body = JSON.stringify({
      count: resultList.length,
      list: commentlist.reverse(),
    });
    return new Response(body, {
      status: 200,
      headers: {
        "Content-Type": "application/json; charset=UTF-8",
        "Access-Control-Allow-Origin": allowedOrigin,
      },
    });
  }

  // handle new comments
  if (req.method === "POST") {
    const newComment = await req.json();

    if (!newComment.author || !newComment.email || !newComment.content) {
      return new Response("名字、邮箱、评论内容不能为空");
    } else if (!isValidUrl(newComment.website)) {
      return new Response("网址格式错误");
    } else {
      const record = await pb.collection("comments").create({
        "uri": newComment.uri,
        "author": newComment.author,
        "email": newComment.email,
        "website": newComment.website,
        "content": newComment.content,
        "parent": newComment.parent,
      });

      const body = JSON.stringify({
        id: record.id,
        author: record.author,
        avatar: new Md5().update(record.email).toString(),
        website: record.website,
        content: record.content,
        created: record.created,
      });
      return new Response(body, {
        status: 200,
        headers: {
          "Content-Type": "application/json; charset=UTF-8",
          "Access-Control-Allow-Origin": allowedOrigin,
        },
      });
    }
  }

  if (req.method === "OPTIONS") {
    return new Response(null, {
      status: 204,
      headers: {
        "Access-Control-Allow-Methods": "GET, POST",
        "Access-Control-Allow-Origin": allowedOrigin,
        "Access-Control-Allow-Headers": "Origin, Referer, Content-Type",
      },
    });
  }

  return new Response("Bad request!");
}

serve(handler);
