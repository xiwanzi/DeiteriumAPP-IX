import React, { useState } from "react";
import {
  ArrowUpRight,
  ShoppingBag,
  ShieldCheck,
  Mail,
  Minus,
  Plus,
  PackageCheck,
} from "lucide-react";
import {
  PageHead,
  SectionHead,
  Tabs,
  ProductCard,
  Price,
  Button,
  Modal,
  Empty,
  media,
} from "./components.jsx";
export default function Store({ state, act, open, navigate, query }) {
  const [brand, setBrand] = useState("全部精选");
  const filtered = state.products.filter(
    (p) =>
      (brand === "全部精选" || p.brand === brand) &&
      `${p.title}${p.subtitle}`.toLowerCase().includes(query.toLowerCase()),
  );
  const add = (p) => act({ type: "CART", id: p.id, delta: 1 }, "已加入购物袋");
  return (
    <>
      <PageHead
        eyebrow="CURATED FOR YOUR WORLD"
        title={
          <>
            好东西，<span className="muted-heading">值得一起发现。</span>
          </>
        }
        subtitle="精选装备与日常灵感，给你的世界多一点可能。"
      >
        <Button secondary onClick={() => open({ type: "cart" })}>
          <ShoppingBag size={17} />
          购物袋
        </Button>
      </PageHead>
      {!query && brand === "全部精选" && (
        <div className="feature-grid">
          <button
            className="feature-card iphone-feature"
            onClick={() =>
              open({
                type: "product",
                item: state.products.find((p) => p.id === "iphone"),
              })
            }
          >
            <img src={media("store_iphone.webp")} alt="橙色 iPhone 17 Pro" />
            <div className="feature-copy">
              <span>本周精选 / APPLE</span>
              <h2>
                出色，
                <br />
                不止一面。
              </h2>
              <p>iPhone 17 Pro</p>
            </div>
            <div className="feature-foot">
              <span>发现本周好物</span>
              <i>
                <ArrowUpRight size={21} />
              </i>
            </div>
          </button>
          <button
            className="feature-card mac-feature"
            onClick={() =>
              open({
                type: "product",
                item: state.products.find((p) => p.id === "macbook"),
              })
            }
          >
            <img src={media("store_macbook.webp")} alt="天蓝色 MacBook Air" />
            <div className="feature-copy">
              <span>灵感装备 / MACBOOK AIR</span>
              <h2>
                轻一点。
                <br />
                想远一点。
              </h2>
              <p>把下一份灵感，带在身边。</p>
            </div>
            <div className="feature-foot">
              <span>探索 MacBook Air</span>
              <i>
                <ArrowUpRight size={21} />
              </i>
            </div>
          </button>
        </div>
      )}
      <div className="store-assurances">
        <span>
          <ShieldCheck size={17} />
          官方精选
        </span>
        <span>
          <Mail size={17} />
          游戏内邮箱交付
        </span>
        <span>
          <PackageCheck size={17} />
          未领取可申请退款
        </span>
        <span className="assurance-note">商品与价格为体验示例</span>
      </div>
      <SectionHead
        title={query ? `“${query}”的搜索结果` : "精选好物"}
        description="有用的，好看的，和你喜欢的。"
      />
      <Tabs
        values={["全部精选", "Apple", "NVIDIA", "AMD"]}
        value={brand}
        onChange={setBrand}
      />
      {filtered.length ? (
        <div className="product-grid">
          {filtered.map((p) => (
            <ProductCard
              key={p.id}
              product={p}
              onAdd={add}
              onOpen={(item) => open({ type: "product", item })}
            />
          ))}
        </div>
      ) : (
        <Empty />
      )}
      <button className="community-banner" onClick={() => navigate("/market")}>
        <img
          src={media("street-daylight.png")}
          alt="Deuterium 玩家与服务器实景"
        />
        <div>
          <span className="eyebrow">MADE BY OUR COMMUNITY</span>
          <h2>好东西，也在玩家手中。</h2>
          <p>逛逛市场，发现大家的创造。</p>
          <span className="banner-link">
            前往玩家市场 <ArrowUpRight size={17} />
          </span>
        </div>
      </button>
    </>
  );
}
export function ProductDetail({ item, state, act, open, close }) {
  const p = state.products.find((x) => x.id === item.id) || item;
  return (
    <Modal title="商品详情" close={close} wide>
      <div className="product-detail">
        <div className={`detail-photo ${p.tone}`}>
          <img src={media(p.image)} alt={p.title} />
        </div>
        <div>
          <span className="tiny-label">DEUTERIUM 官方商城 · {p.brand}</span>
          <h2>{p.title}</h2>
          <p className="description">{p.subtitle}</p>
          <Price value={p.price} />
          <dl className="detail-list">
            <div>
              <dt>交付方式</dt>
              <dd>游戏内邮箱</dd>
            </div>
            <div>
              <dt>剩余库存</dt>
              <dd>{p.stock} 件</dd>
            </div>
            <div>
              <dt>退款说明</dt>
              <dd>未领取可退，领取后不可退</dd>
            </div>
          </dl>
          <p className="notice-box">
            沿用 App 的商品展示素材。本机体验不售卖实体硬件，不发放游戏物品。
          </p>
          <Button
            disabled={!p.stock}
            onClick={() => {
              if (act({ type: "CART", id: p.id, delta: 1 }, "已加入购物袋"))
                open({ type: "cart" });
            }}
          >
            <ShoppingBag size={17} />
            {p.stock ? "加入购物袋" : "暂时售罄"}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
export function Cart({ state, act, open, close }) {
  const entries = Object.entries(state.cart[state.user] || {}).map(
    ([pid, qty]) => ({ ...state.products.find((p) => p.id === pid), qty }),
  );
  const total = entries.reduce((n, p) => n + p.price * p.qty, 0);
  return (
    <Modal title="购物袋" close={close}>
      {entries.length ? (
        <>
          <div className="cart-items">
            {entries.map((p) => (
              <div key={p.id} className="cart-item">
                <img src={media(p.image)} alt={p.title} />
                <div>
                  <h3>{p.title}</h3>
                  <Price small value={p.price} />
                  <div className="stepper">
                    <button
                      aria-label={`减少 ${p.title}`}
                      onClick={() => act({ type: "CART", id: p.id, delta: -1 })}
                    >
                      <Minus size={15} />
                    </button>
                    <span>{p.qty}</span>
                    <button
                      aria-label={`增加 ${p.title}`}
                      onClick={() => act({ type: "CART", id: p.id, delta: 1 })}
                    >
                      <Plus size={15} />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
          <div className="cart-total">
            <span>合计</span>
            <Price value={total} />
          </div>
          <Button
            className="full"
            onClick={() =>
              open({
                type: "confirm",
                title: "确认商城订单",
                description: "确认商品与数量后，使用演示信用点付款。",
                amount: total,
                action: { type: "CHECKOUT" },
                success: "订单已创建，可在我的订单查看",
              })
            }
          >
            确认结算
          </Button>
        </>
      ) : (
        <Empty title="购物袋还是空的" text="把喜欢的好物放进来吧。">
          <Button onClick={close}>继续逛逛</Button>
        </Empty>
      )}
    </Modal>
  );
}
